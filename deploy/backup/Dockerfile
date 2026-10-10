# Nightly encrypted PostgreSQL dumps, copied off-host. See doc/operations.md.
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
RUN apk add --no-cache age openssh-client postgresql16-client rsync \
 && adduser -D -u 1000 backup
COPY backup.sh /usr/local/bin/axon-backup
USER backup
ENTRYPOINT ["axon-backup"]
CMD ["schedule"]
