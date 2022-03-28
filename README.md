# go-httpproxy

[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/isayme/httpproxy?sort=semver&style=flat-square)](https://hub.docker.com/r/isayme/httpproxy)
![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/isayme/httpproxy?sort=semver&style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/isayme/httpproxy?style=flat-square)

A simple http proxy, support HTTP/HTTPS/HTTP2/Websocket.

# Docker Compose

```
version: "3"

services:
  httpproxy:
    container_name: httpproxy
    image: isayme/httpproxy:latest
    ports:
      - "1087:1087"
    command: /app/httpproxy -p 1087
    # command: httpproxy --proxy socks5://your-host:your-port -p 1087
    # command: httpproxy --proxy http://your-host:your-port -p 1087
    # command: httpproxy --proxy https://your-host:your-port -p 1087
```

# Proxy Protocol Screenshoot

## for HTTP

![HTTP Protocol](./doc/http.png)

## for HTTPS

![HTTPS Protocol](./doc/https.png)

# Refers

- [HTTP 代理原理及实现（一）](https://imququ.com/post/web-proxy.html)
