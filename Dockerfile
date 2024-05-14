FROM golang:1.22-alpine as builder
WORKDIR /app

ARG APP_NAME
ENV APP_NAME ${APP_NAME}
ARG APP_VERSION
ENV APP_VERSION ${APP_VERSION}

COPY . .
RUN mkdir -p ./dist  \
  && GO111MODULE=on GOPROXY=https://goproxy.io,direct go mod download \
  && go build -ldflags "-X github.com/isayme/go-httpproxy/httpproxy.Name=${APP_NAME} \
  -X github.com/isayme/go-httpproxy/httpproxy.Version=${APP_VERSION}" \
  -o ./dist/httpproxy main.go

FROM alpine
WORKDIR /app

ARG APP_NAME
ENV APP_NAME ${APP_NAME}
ARG APP_VERSION
ENV APP_VERSION ${APP_VERSION}

COPY --from=builder /app/dist/httpproxy ./

CMD ["/app/httpproxy"]
