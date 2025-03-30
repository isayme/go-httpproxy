# httpproxy

[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/isayme/httpproxy?sort=semver&style=flat-square)](https://hub.docker.com/r/isayme/httpproxy)
![Docker Image Size (latest semver)](https://img.shields.io/docker/image-size/isayme/httpproxy?sort=semver&style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/isayme/httpproxy?style=flat-square)

A simple http & https proxy server.

# Useage

## Docker

```
docker run --rm -p 1087:1087 isayme/httpproxy
```

## Docker Compose

```
version: "3"

services:
  httpproxy:
    container_name: httpproxy
    image: isayme/httpproxy:latest
    ports:
      - "1087:1087"
    # run as http proxy, port 1087
    command: /app/httpproxy -p 1087

    # run as http proxy, port 1087, request remote with another socks5 proxy
    # command: httpproxy --proxy socks5://your-host:your-port -p 1087

    # run as http proxy, port 1087, request remote with another http proxy
    # command: httpproxy --proxy http://your-host:your-port -p 1087

    # run as http proxy, port 1087, request remote with another https proxy
    # command: httpproxy --proxy https://your-host:your-port -p 1087
```

# Refers

- [HTTP 代理原理及实现（一）](https://imququ.com/post/web-proxy.html)
