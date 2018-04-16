FROM alpine

RUN apk --update upgrade && \
    apk add --no-cache openssl curl ca-certificates && \
    rm -rf /var/cache/apk/*

COPY build/move.sh /move.sh
COPY dist/bin/oci /oci

CMD ["/move.sh"]
