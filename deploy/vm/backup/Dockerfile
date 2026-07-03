FROM alpine:3.20

RUN apk add --no-cache postgresql16-client rclone ca-certificates \
    && wget -qO /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-amd64/mc \
    && chmod +x /usr/local/bin/mc \
    && adduser -D -u 10001 kmap

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

USER kmap
ENTRYPOINT ["/usr/local/bin/backup.sh"]
