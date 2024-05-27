package httpproxy

import "time"

type serverOptions struct {
	listenPort    uint16
	listenAddress string

	username string
	password string

	proxy          string
	connectTimeout time.Duration
	timeout        time.Duration

	certFile string
	keyFile  string

	pretendAsWeb bool
}

type ServerOption interface {
	apply(*serverOptions)
}

type funcServerOption struct {
	f func(*serverOptions)
}

func (fdo funcServerOption) apply(do *serverOptions) {
	fdo.f(do)
}

func newFuncServerOption(f func(*serverOptions)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}

func WithListenPort(listenPort uint16) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.listenPort = listenPort
	})
}

func WithListenAddress(listenAddress string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.listenAddress = listenAddress
	})
}

func WithUsername(username string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.username = username
	})
}

func WithPassword(password string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.password = password
	})
}

func WithProxy(addr string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.proxy = addr
	})
}

func WithConnectTimeout(timeout time.Duration) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.connectTimeout = timeout
	})
}

func WithTimeout(timeout time.Duration) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.timeout = timeout
	})
}

func WithCertFile(certFile string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.certFile = certFile
	})
}

func WithKeyFile(keyFile string) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.keyFile = keyFile
	})
}

func WithPretendAsWeb(pretendAsWeb bool) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.pretendAsWeb = pretendAsWeb
	})
}
