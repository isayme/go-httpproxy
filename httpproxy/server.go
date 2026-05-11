package httpproxy

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/isayme/go-logger"
	"golang.org/x/net/proxy"
)

var responseConnectionEstablished = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

type Server struct {
	dialer proxy.ContextDialer

	options serverOptions

	httpServer *http.Server
}

func NewServer(opts ...ServerOption) (*Server, error) {
	s := &Server{
		dialer: proxy.Direct,
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			opt.apply(&s.options)
		}
	}

	proxyAddress := s.options.proxy
	if proxyAddress != "" {
		url, err := url.Parse(proxyAddress)
		if err != nil {
			return nil, fmt.Errorf("NewServer: parse proxy address fail: %w", err)
		}

		dialer, err := proxy.FromURL(url, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("NewServer: create proxy dialer fail: %w", err)
		}

		s.dialer = NewProxyContextDialer(dialer)
	}

	return s, nil
}

func (s *Server) dial(network, addr string) (c net.Conn, err error) {
	ctx := context.Background()
	if s.options.connectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.options.connectTimeout)
		defer cancel()
	}

	return s.dialer.DialContext(ctx, network, addr)
}

func (s *Server) ListenAndServe() error {
	address := s.options.listenAddress
	if address == "" {
		address = fmt.Sprintf(":%d", s.options.listenPort)
	}

	s.httpServer = &http.Server{
		Addr:    address,
		Handler: s,
	}

	certFile := s.options.certFile
	keyFile := s.options.keyFile
	if certFile != "" && keyFile != "" {
		logger.Infow("start listen with tls ...", "addr", address)
		return s.httpServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		logger.Infow("start listen ...", "addr", address)
		return s.httpServer.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	seqId := randSeqId()

	if r.Method == http.MethodConnect {
		if r.URL.Port() == "" {
			r.URL.Host = fmt.Sprintf("%s:%d", r.URL.Host, 443)
		}
	} else {
		if r.URL.Port() == "" {
			r.URL.Host = fmt.Sprintf("%s:%d", r.URL.Host, 80)
		}
	}

	// not proxy request, response version
	if r.URL.Hostname() == "" {
		if s.options.pretendAsWeb {
			w.WriteHeader(404)
			w.Write([]byte("404 page not found\n"))
			return
		}

		w.Write([]byte(fmt.Sprintf("Server1: %s\n", Name)))
		w.Write([]byte(fmt.Sprintf("Server1: %s\n", r.URL.String())))
		w.Write([]byte(fmt.Sprintf("%s %s\n", Name, Version)))
		return
	}

	// auth
	if s.options.username != "" && s.options.password != "" {
		authorization := r.Header.Get("Proxy-Authorization")
		username, password, ok := parseBasicAuth(authorization)
		if !ok || username != s.options.username || password != s.options.password {
			if s.options.pretendAsWeb {
				w.WriteHeader(404)
				w.Write([]byte("404 page not found\n"))
				return
			}

			w.WriteHeader(407)
			w.Header().Add("Content-Type", "text/plain")
			w.Write([]byte(fmt.Sprintf("Server2: %s\n", Name)))
			w.Write([]byte(fmt.Sprintf("%s %s\n", Name, Version)))
			return
		}
	}

	logger.Infow("newRequest", "url", r.URL.String(), "client", r.RemoteAddr, "seqId", seqId, "host", r.Host)
	start := time.Now()
	defer func() {
		logger.Infow("handleRequest", "url", r.URL.String(), "duration", time.Since(start).String(), "seqId", seqId)
	}()

	remoteConn, err := s.dial("tcp", r.URL.Host)
	if err != nil {
		logger.Warnw("dial remote fail", "err", err, "addr", r.URL.Host, "seqId", seqId)
		return
	}
	logger.Debugw("dial remote ok", "addr", r.URL.Host, "remote", remoteConn.RemoteAddr().String(), "seqId", seqId)

	defer remoteConn.Close()
	tcpRemoteConn, _ := remoteConn.(*net.TCPConn)
	remoteConn = NewTimeoutConn(remoteConn, s.options.timeout)

	whj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, _, err := whj.Hijack()
	if err != nil {
		logger.Warnw("hijack fail", "err", err, "seqId", seqId)
		return
	}

	if r.Method == http.MethodConnect {
		// response ok
		_, err := conn.Write(responseConnectionEstablished)
		if err != nil {
			logger.Warnw("https resopnse 200 fail", "err", err, "seqId", seqId)
			return
		}
		logger.Debugw("write to client connection established ok", "addr", r.URL.Host, "seqId", seqId)
	} else {
		// write request data to remote
		err = r.Write(remoteConn)
		if err != nil {
			logger.Warnw("remote write line fail", "err", err, "seqId", seqId)
			return
		}
		logger.Debugw("write req to remote ok", "addr", r.URL.Host, "seqId", seqId)
	}

	closeWriter := mustGetWriteCloser(conn)
	conn = NewTimeoutConn(conn, s.options.timeout)

	// see https://stackoverflow.com/a/75418345/1918831
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		var err error
		var n int64
		n, err = io.Copy(remoteConn, conn)
		logger.Debugw("copy from client end", "addr", r.URL.Host, "n", n, "err", err, "seqId", seqId)
		tcpRemoteConn.CloseWrite()
	}()

	go func() {
		defer wg.Done()

		var err error
		var n int64
		n, err = io.Copy(conn, remoteConn)
		logger.Debugw("copy from remote end", "addr", r.URL.Host, "n", n, "err", err, "seqId", seqId)
		closeWriter.CloseWrite()
	}()

	wg.Wait()
}

// from package http
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}
