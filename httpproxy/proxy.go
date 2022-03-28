package httpproxy

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("http", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return NewHttpProxy(u, d)
	})
	proxy.RegisterDialerType("https", func(u *url.URL, d proxy.Dialer) (proxy.Dialer, error) {
		return NewHttpProxy(u, d)
	})
}

type HttpProxy struct {
	u *url.URL
	d proxy.Dialer
}

type httpDialer struct {
}

func (d httpDialer) Dial(network, addr string) (net.Conn, error) {
	return net.Dial(network, addr)
}

type httpsDialer struct {
}

func (d httpsDialer) Dial(network, addr string) (net.Conn, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	return tls.Dial(network, addr, conf)
}

func NewHttpProxy(u *url.URL, d proxy.Dialer) (*HttpProxy, error) {
	switch u.Scheme {
	case "http":
		d = httpDialer{}
	case "https":
		d = httpsDialer{}
	default:
		return nil, fmt.Errorf("schema '%s' invalid", u.Scheme)
	}

	return &HttpProxy{
		u: u,
		d: d,
	}, nil
}

func (hp *HttpProxy) Dial(network, addr string) (c net.Conn, err error) {
	defer func() {
		if err != nil && c != nil {
			c.Close()
		}
	}()

	c, err = hp.d.Dial("tcp", hp.u.Host)
	if err != nil {
		return nil, err
	}

	req := http.Request{
		Method: "CONNECT",
		Host:   hp.u.Host,
		Header: http.Header{},
		URL: &url.URL{
			Path: addr,
		},
	}

	if hp.u.User != nil {
		password, _ := hp.u.User.Password()
		auth := fmt.Sprintf("%s:%s", hp.u.User.Username(), password)
		req.Header.Add("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	err = req.Write(c)
	if err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != 200 {
		return nil, fmt.Errorf("connect get stausCode %d", statusCode)
	}

	return
}
