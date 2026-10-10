# Nightly encrypted PostgreSQL dumps, copied off-host. See doc/operations.md.
FROM postgres:16-alpine
RUN apk add --no-cache age openssh-client rsync \
 && adduser -D -u 1000 backup
COPY backup.sh /usr/local/bin/axon-backup
USER backup
ENTRYPOINT ["axon-backup"]
CMD ["schedule"]
