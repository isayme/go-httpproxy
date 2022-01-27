# go-httpproxy

[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/isayme/httpproxy?sort=semver&style=flat-square)](https://hub.docker.com/r/isayme/httpproxy)
![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/isayme/httpproxy?sort=semver&style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/isayme/httpproxy?style=flat-square)

A simple http proxy, support HTTP/HTTPS/HTTP2/Websocket.

# Run with Cli

```
// simple proxy
httpproxy -p 1087

// proxy with socks5
httpproxy --proxy socks5://your-host:1080 -p 1087
```

# Docker Compose
```
version: "3"

services:
  httpproxy:
    container_name: httpproxy
    image: isayme/httpproxy:latest
    ports:
      - "1087:1087"
    command: /app/httpproxy --proxy socks5://{socks5 server addr}:{socks5 server port} -p 1087
```

# Proxy Protocol Screenshoot

## for HTTP

![HTTP Protocol](./doc/http.png)

## for HTTPS

![HTTPS Protocol](./doc/https.png)

# Refers

- [HTTP 代理原理及实现（一）](https://imququ.com/post/web-proxy.html)
