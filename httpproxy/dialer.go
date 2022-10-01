package httpproxy

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

type ProxyContextDialer struct {
	d proxy.Dialer
}

func NewProxyContextDialer(d proxy.Dialer) ProxyContextDialer {
	return ProxyContextDialer{
		d: d,
	}
}

/**
 * from golang.org/x/net/proxy.dialContext
 */
func (d ProxyContextDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var (
		conn net.Conn
		done = make(chan struct{}, 1)
		err  error
	)
	go func() {
		conn, err = d.d.Dial(network, address)
		close(done)
		if conn != nil && ctx.Err() != nil {
			conn.Close()
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-done:
	}
	return conn, err
}
