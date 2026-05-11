package httpproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/isayme/go-logger"
	"github.com/stretchr/testify/require"
)

func createProxy(require *require.Assertions, opts ...ServerOption) (chan struct{}, func()) {
	proxy, err := NewServer(opts...)
	require.Nil(err)

	address := proxy.options.listenAddress
	if address == "" {
		address = fmt.Sprintf(":%d", proxy.options.listenPort)
	}

	proxy.httpServer = &http.Server{
		Addr:    address,
		Handler: proxy,
	}

	ch := make(chan struct{}, 1)
	ln, err := net.Listen("tcp", address)
	require.Nil(err)

	ch <- struct{}{}

	certFile := proxy.options.certFile
	keyFile := proxy.options.keyFile
	if certFile != "" && keyFile != "" {
		logger.Infow("start listen with tls ...", "addr", address)
		go proxy.httpServer.ServeTLS(ln, certFile, keyFile)
	} else {
		logger.Infow("start listen ...", "addr", address)
		go proxy.httpServer.Serve(ln)
	}

	return ch, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		proxy.Shutdown(ctx)
	}
}

func TestHTTP(t *testing.T) {
	require := require.New(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(400)
		w.Write([]byte("hello upstream"))
	}))
	defer upstream.Close()

	ch, stop := createProxy(require, WithListenAddress(":8080"), WithPretendAsWeb(true))
	defer stop()
	<-ch

	proxyUrl, err := url.Parse("http://127.0.0.1:8080")
	require.Nil(err)

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(proxyUrl),
		},
	}

	resp, err := client.Get(upstream.URL)
	require.Nil(err)

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	require.Equal(400, resp.StatusCode)
	require.Equal("hello upstream", string(body))
	require.Equal("ok", resp.Header.Get("x-upstream"))
}

func TestHTTPS(t *testing.T) {
	require := require.New(t)

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(400)
		w.Write([]byte("hello upstream"))
	}))
	defer upstream.Close()

	ch, stop := createProxy(require, WithListenAddress(":8080"), WithPretendAsWeb(true))
	defer stop()
	<-ch

	proxyUrl, err := url.Parse("http://127.0.0.1:8080")
	require.Nil(err)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxyUrl),
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get(upstream.URL)
	require.Nil(err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(400, resp.StatusCode)
	require.Equal("hello upstream", string(body))
	require.Equal("ok", resp.Header.Get("x-upstream"))
}

func TestTCP(t *testing.T) {
	require := require.New(t)

	// upstream server
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(err)
	defer echoLn.Close()

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}

			io.Copy(conn, conn)
			// tcpConn, _ := conn.(*net.TCPConn)
			conn.Close()
			logger.Info("echo handle end")
		}
	}()

	// proxy server
	ch, stop := createProxy(require, WithListenAddress(":8080"), WithPretendAsWeb(true))
	defer stop()
	<-ch

	// client
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	require.Nil(err)
	defer conn.Close()

	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", echoLn.Addr().String(), echoLn.Addr().String())
	_, err = conn.Write([]byte(req))
	require.Nil(err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{})
	require.Nil(err)
	defer resp.Body.Close()
	require.Equal(200, resp.StatusCode)

	// send data
	_, err = conn.Write([]byte("hello upstream"))
	require.Nil(err)

	// read data
	time.Sleep(time.Second)
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.Nil(err)
	require.Equal("hello upstream", string(buf[:n]))
}

func TestMock1(t *testing.T) {
	require := require.New(t)

	// proxy server
	ch, stop := createProxy(require, WithListenAddress(":8080"), WithPretendAsWeb(true))
	defer stop()
	<-ch

	// client
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	require.Nil(err)
	defer conn.Close()

	// req := fmt.Sprintf("GET %s HTTP/1.1\r\n\r\n\r\n", echoLn.Addr().String())
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\n\r\n", "/", "baidu.com")
	_, err = conn.Write([]byte(req))
	require.Nil(err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{})
	require.Nil(err)
	defer resp.Body.Close()

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	require.Equal("404 page not found\n", string(buf[:n]))

	require.Equal(404, resp.StatusCode)
}

func TestMock2(t *testing.T) {
	require := require.New(t)

	ch, stop := createProxy(require, WithListenAddress(":8080"), WithPretendAsWeb(true))
	defer stop()
	<-ch

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	resp, err := client.Get("http://127.0.0.1:8080")
	require.Nil(err)

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	require.Equal(404, resp.StatusCode)
	require.Equal("404 page not found\n", string(body))
}
