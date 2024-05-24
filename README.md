# httpproxy

[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/isayme/httpproxy?sort=semver&style=flat-square)](https://hub.docker.com/r/isayme/httpproxy)
![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/isayme/httpproxy?sort=semver&style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/isayme/httpproxy?style=flat-square)

A simple http & https proxy server.

# Useage

## Server: Docker Compose

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

## Client

> curl -x http://127.0.0.1:1087 http://baidu.com

# Refers

- [HTTP 代理原理及实现（一）](https://imququ.com/post/web-proxy.html)
