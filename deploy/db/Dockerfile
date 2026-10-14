# syntax=docker/dockerfile:1
# PostgreSQL 16 with pgvector, plus Debian's security updates: the upstream
# image is rebuilt less often than Debian patches its packages (perl, pcre2…).
# Same PostgreSQL major version and Debian release, so the data directory
# and collations are unchanged. gosu, which the entrypoint uses once to drop
# from root to postgres, is rebuilt with the patched Go toolchain the API uses.
FROM golang:1.26-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade AS gosu
ARG GOSU_VERSION=1.19
RUN CGO_ENABLED=0 GOBIN=/out go install -trimpath -ldflags '-s -w' github.com/tianon/gosu@${GOSU_VERSION}

FROM pgvector/pgvector:pg16@sha256:7b822b0aac60967beb1ea5e576b8602c94c300a157d187f385ae3e0da199b90a
RUN apt-get update && apt-get -y upgrade && rm -rf /var/lib/apt/lists/*
COPY --from=gosu /out/gosu /usr/local/bin/gosu
