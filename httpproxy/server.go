package httpproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/isayme/go-logger"
	"golang.org/x/net/proxy"
)

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
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	line, err := rw.ReadString('\n')
	if err != nil {
		logger.Warnf("read request info fail: %v", err)
		return
	}

	result := strings.SplitN(line, " ", 3)
	if len(result) <= 2 {
		logger.Warnw("not valid request info")
		return
	}
	method := result[0]
	rawurl := result[1]

	// https
	if method == http.MethodConnect {
		if strings.Index(rawurl, "://") < 0 {
			rawurl = fmt.Sprintf("https://%s", rawurl)
		}
	}

	url, err := url.Parse(rawurl)
	if err != nil {
		logger.Warnw("url.Parse request url fail", "err", err)
		return
	}

	if method == http.MethodConnect {
		if url.Port() == "" {
			url.Host = fmt.Sprintf("%s:%d", url.Host, 443)
		}
	} else {
		if url.Port() == "" {
			url.Host = fmt.Sprintf("%s:%d", url.Host, 80)
		}
	}

	logger.Infow("newRequest", "url", url.String())
	remoteConn, err := s.dial("tcp", url.Host)
	if err != nil {
		logger.Warnw("dial remote fail", "err", err, "addr", url.Host)
		return
	}
	defer remoteConn.Close()

	if method == http.MethodConnect {
		// response ok
		_, err := rw.WriteString("HTTP/1.1 200 Connection established\r\n\r\n")
		if err != nil {
			logger.Warnf("https resopnse 200 fail", "err", err)
			return
		}
		rw.Flush()

		// reset to ignore CONNECT request data
		rw.Reader.Reset(conn)
	} else {
		_, err = remoteConn.Write([]byte(line))
		if err != nil {
			logger.Warnf("remote write line fail", "err", err)
			return
		}
	}

	go io.Copy(rw, remoteConn)
	io.Copy(remoteConn, rw)
}
