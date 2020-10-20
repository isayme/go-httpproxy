package httpproxy

import (
	"bufio"
	"bytes"
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
	address      string
	proxyAddress string
	dialer       proxy.Dialer
}

func NewServer(address string, proxyAddress string) (*Server, error) {
	s := &Server{
		address:      address,
		proxyAddress: proxyAddress,
	}

	if proxyAddress != "" {
		url, err := url.Parse(proxyAddress)
		if err != nil {
			return nil, fmt.Errorf("NewServer: parse proxy address fail: %w", err)
		}

		dialer, err := proxy.FromURL(url, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("NewServer: create proxy dialer fail: %w", err)
		}

		s.dialer = dialer
	}

	return s, nil
}

func (s *Server) dial(network, addr string) (c net.Conn, err error) {
	if s.dialer == nil {
		return net.Dial(network, addr)
	}

	return s.dialer.Dial(network, addr)
}

func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		logger.Errorf("net.Listen fail: %v", err)
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Warnf("l.Accept fail: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// read request info
	reqBuf := bytes.NewBuffer(nil)
	rr := io.TeeReader(conn, reqBuf)
	req, err := http.ReadRequest(bufio.NewReader(rr))
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
		_, err = remoteConn.Write(reqBuf.Bytes())
		if err != nil {
			logger.Warnf("remote write line fail", "err", err)
			return
		}
	}

	go io.Copy(conn, remoteConn)
	io.Copy(remoteConn, conn)
}
