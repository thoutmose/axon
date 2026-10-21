# syntax=docker/dockerfile:1
# PostgreSQL 16 with pgvector, plus Debian's security updates: the upstream
# image is rebuilt less often than Debian patches its packages (perl, pcre2…).
# Same PostgreSQL major version and Debian release, so the data directory
# and collations are unchanged. gosu, which the entrypoint uses once to drop
# from root to postgres, is rebuilt with the patched Go toolchain the API uses.
FROM golang:1.26-alpine@sha256:9ce231b28998b32e116927c61aa5c406183061d4f3f2d61b292c88e7f6bdf7ca AS gosu
ARG GOSU_VERSION=1.19
RUN CGO_ENABLED=0 GOBIN=/out go install -trimpath -ldflags '-s -w' github.com/tianon/gosu@${GOSU_VERSION}

FROM pgvector/pgvector:pg16@sha256:7b822b0aac60967beb1ea5e576b8602c94c300a157d187f385ae3e0da199b90a
# gnupg only served to add PostgreSQL's apt key: it goes, and SQLite with it.
RUN apt-get update && apt-get -y upgrade && apt-get -y purge --auto-remove gpg && rm -rf /var/lib/apt/lists/*
COPY --from=gosu /out/gosu /usr/local/bin/gosu
