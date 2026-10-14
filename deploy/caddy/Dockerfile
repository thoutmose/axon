# syntax=docker/dockerfile:1
# Caddy built with the patched Go toolchain the API uses: the upstream image
# can lag behind Go security releases (denial-of-service fixes in net/http,
# HTTP/2, and TLS). Same Caddy version and standard modules as upstream.
FROM golang:1.26-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade AS build
ARG CADDY_VERSION=v2.11.7
ARG XNET_VERSION=v0.60.0
WORKDIR /src
RUN printf 'package main\n\nimport (\n\tcaddycmd "github.com/caddyserver/caddy/v2/cmd"\n\t_ "github.com/caddyserver/caddy/v2/modules/standard"\n)\n\nfunc main() { caddycmd.Main() }\n' > main.go \
    && go mod init caddy \
    && go get github.com/caddyserver/caddy/v2@${CADDY_VERSION} golang.org/x/net@${XNET_VERSION} \
    && go mod tidy \
    && CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o /caddy .

FROM caddy:2-alpine@sha256:d8542f48d34a9cf4e4c11a478865229840e87e4c96ea3f439101f31a5d35f75f
COPY --from=build /caddy /usr/bin/caddy
