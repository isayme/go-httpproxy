package httpproxy

import (
	"crypto/tls"
	"net"
)

type CloseWriter interface {
	CloseWrite() error
}

func mustGetWriteCloser(conn net.Conn) CloseWriter {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		return tcpConn
	}

	if tlsConn, ok := conn.(*tls.Conn); ok {
		return tlsConn
	}

	panic("conn not convert to WriteCloser")
}
