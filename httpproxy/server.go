package httpproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/isayme/go-logger"
	"golang.org/x/net/proxy"
)

var responseOk = []byte("HTTP/1.1 200 OK\r\n")
var responseConnectionEstablished = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

type Server struct {
	address string
	dialer  proxy.ContextDialer

	options serverOptions
}

func NewServer(address string, opts ...ServerOption) (*Server, error) {
	s := &Server{
		address:      address,
		dialer:  proxy.Direct,
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

func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		logger.Errorf("net.Listen fail: %v", err)
		return err
	}

	s.Serve(l)
	return nil
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	l, err := tls.Listen("tcp", s.address, tlsConfig)
	if err != nil {
		logger.Errorf("net.Listen fail: %v", err)
		return err
	}

	s.Serve(l)
	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		logger.Warnw("http.ReadRequest fail", "err", err)
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
		conn.Write(responseOk)
		conn.Write([]byte("Content-Type: text/plain\r\n"))
		conn.Write([]byte(fmt.Sprintf("Server: %s\r\n\r\n", Name)))
		conn.Write([]byte(fmt.Sprintf("%s %s\r\n\r\n", Name, Version)))
		return
	}

	logger.Infow("newRequest", "url", req.URL.String())
	remoteConn, err := s.dial("tcp", req.URL.Host)
	if err != nil {
		logger.Warnw("dial remote fail", "err", err, "addr", req.URL.Host)
		return
	}
	defer remoteConn.Close()

	if req.Method == http.MethodConnect {
		// response ok
		_, err := conn.Write(responseConnectionEstablished)
		if err != nil {
			logger.Warnf("https resopnse 200 fail", "err", err)
			return
		}
	} else {
		// write request data to remote
		err = req.Write(remoteConn)
		if err != nil {
			logger.Warnf("remote write line fail", "err", err)
			return
		}
	}

	go func() {
		io.Copy(conn, remoteConn)
		conn.Close()
	}()

	io.Copy(remoteConn, conn)
}
