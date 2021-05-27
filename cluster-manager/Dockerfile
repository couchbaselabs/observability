FROM alpine:3.13 as cert-generator
RUN apk --update --no-cache add openssl

RUN mkdir -p /certs
WORKDIR /certs
RUN openssl req -x509 -nodes -days 365 -subj "/C=CA/ST=QC/O=Couchbase, Inc./CN=couchbase.com" -addext "subjectAltName=DNS:couchbase.com" -newkey rsa:2048 -keyout /certs/server.key -out /certs/server.crt;

FROM golang:1.16 as builder

RUN apt-get update && \
    apt-get install -y sqlcipher libssl-dev openssl && \
    rm -rf /var/lib/apt/lists/*

COPY repo/ /src/github.com/couchbaselabs/cbmultimanager/
WORKDIR /src/github.com/couchbaselabs/cbmultimanager
RUN go mod download && mkdir -p /bin
# Statically build it properly
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o /bin/cbmultimanager ./cmd/cbmultimanager

FROM alpine:3.13
RUN apk add --no-cache tini bash curl
ENV TINI_SUBREAPER=

RUN mkdir -p /data /priv
COPY --from=cert-generator /certs/ /priv/

COPY --from=builder /bin/cbmultimanager /bin/cbmultimanager
RUN chmod a+x /bin/cbmultimanager

COPY entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh

EXPOSE 7196 7197
ENTRYPOINT [ "/sbin/tini", "--", "/entrypoint.sh" ]
CMD [ "--sqlite-db", "/data/data.sqlite", "--sqlite-key", "password", "--cert-path", "/priv/server.crt", "--key-path", "/priv/server.key", "-log-level", "debug" ]