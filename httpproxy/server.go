package httpproxy

import (
	"bufio"
	"context"
	"crypto/tls"
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

var responseOk = []byte("HTTP/1.1 200 OK\r\n")
var responseNotFound = []byte("HTTP/1.1 404 Not Found\r\n")
var responseConnectionEstablished = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

type Server struct {
	dialer proxy.ContextDialer

	options serverOptions
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

func (s *Server) Serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Warnf("l.Accept fail: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) Listen() (net.Listener, error) {
	address := s.options.listenAddress
	if address == "" {
		address = fmt.Sprintf(":%d", s.options.listenPort)
	}

	certFile := s.options.certFile
	keyFile := s.options.keyFile
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		logger.Infow("start listen with tls ...", "addr", address)
		return tls.Listen("tcp", address, tlsConfig)
	} else {
		logger.Infow("start listen ...", "addr", address)
		return net.Listen("tcp", address)
	}
}

func (s *Server) ListenAndServe() error {
	l, err := s.Listen()
	if err != nil {
		logger.Errorf("net.Listen fail: %v", err)
		return err
	}

	s.Serve(l)
	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	seqId := randSeqId()

	defer conn.Close()

	closeWriter := mustGetWriteCloser(conn)
	conn = NewTimeoutConn(conn, s.options.timeout)

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		logger.Warnw("http.ReadRequest fail", "err", err, "seqId", seqId)
		return
	}

	if req.Method == http.MethodConnect {
		if req.URL.Port() == "" {
			req.URL.Host = fmt.Sprintf("%s:%d", req.URL.Host, 443)
		}
	} else {
		if req.URL.Port() == "" {
			req.URL.Host = fmt.Sprintf("%s:%d", req.URL.Host, 80)
		}
	}

	// not proxy request, response version
	if req.URL.Hostname() == "" {
		if s.options.pretendAsWeb {
			s.pretendAsWeb(conn)
			return
		}
		conn.Write(responseOk)
		conn.Write([]byte("Content-Type: text/plain\r\n"))
		conn.Write([]byte(fmt.Sprintf("Server: %s\r\n\r\n", Name)))
		conn.Write([]byte(fmt.Sprintf("%s %s\r\n\r\n", Name, Version)))
		return
	}

	// auth
	if s.options.username != "" && s.options.password != "" {
		authorization := req.Header.Get("Proxy-Authorization")
		username, password, ok := parseBasicAuth(authorization)
		if !ok || username != s.options.username || password != s.options.password {
			if s.options.pretendAsWeb {
				s.pretendAsWeb(conn)
				return
			}

			conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n"))
			conn.Write([]byte("Content-Type: text/plain\r\n"))
			conn.Write([]byte(fmt.Sprintf("Server: %s\r\n\r\n", Name)))
			conn.Write([]byte(fmt.Sprintf("%s %s\r\n\r\n", Name, Version)))
			return
		}
	}

	logger.Infow("newRequest", "url", req.URL.String(), "client", conn.RemoteAddr().String(), "seqId", seqId)
	start := time.Now()
	defer func() {
		logger.Infow("handleRequest", "url", req.URL.String(), "duration", time.Since(start).String(), "seqId", seqId)
	}()

	remoteConn, err := s.dial("tcp", req.URL.Host)
	if err != nil {
		logger.Warnw("dial remote fail", "err", err, "addr", req.URL.Host, "seqId", seqId)
		return
	}
	logger.Debugw("dial remote ok", "addr", req.URL.Host, "remote", remoteConn.RemoteAddr().String(), "seqId", seqId)

	defer remoteConn.Close()
	tcpRemoteConn, _ := remoteConn.(*net.TCPConn)
	remoteConn = NewTimeoutConn(remoteConn, s.options.timeout)

	if req.Method == http.MethodConnect {
		// response ok
		_, err := conn.Write(responseConnectionEstablished)
		if err != nil {
			logger.Warnw("https resopnse 200 fail", "err", err, "seqId", seqId)
			return
		}
		logger.Debugw("write to client connection established ok", "addr", req.URL.Host, "seqId", seqId)
	} else {
		// write request data to remote
		err = req.Write(remoteConn)
		if err != nil {
			logger.Warnw("remote write line fail", "err", err, "seqId", seqId)
			return
		}
		logger.Debugw("write req to remote ok", "addr", req.URL.Host, "seqId", seqId)
	}

	// see https://stackoverflow.com/a/75418345/1918831
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		var err error
		var n int64
		n, err = io.Copy(remoteConn, conn)
		logger.Debugw("copy from client end", "addr", req.URL.Host, "n", n, "err", err, "seqId", seqId)
		tcpRemoteConn.CloseWrite()
	}()

	go func() {
		defer wg.Done()

		var err error
		var n int64
		n, err = io.Copy(conn, remoteConn)
		logger.Debugw("copy from remote end", "addr", req.URL.Host, "n", n, "err", err, "seqId", seqId)
		closeWriter.CloseWrite()
	}()

	wg.Wait()
}

func (s *Server) pretendAsWeb(conn net.Conn) {
	conn.Write(responseNotFound)
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte("Content-Length: 19\r\n\r\n"))
	conn.Write([]byte("404 page not found\n"))
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
