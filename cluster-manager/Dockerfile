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
ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN go mod download
RUN go build -o /bin/cbmultimanager ./cmd/cbmultimanager
RUN go build -o /bin/cbeventlog ./cmd/cbeventlog

# For Alpine we need to look at the sqlcipher dependency and slim things down to the minimum
# FROM alpine:3.13
# COPY --from=builder /bin/cbmultimanager /bin/cbmultimanager
# RUN chmod a+x /bin/cbmultimanager

# RUN apk add --no-cache openssl sqlcipher && \

RUN mkdir -p /data /priv
COPY --from=cert-generator /certs/ /priv/

EXPOSE 7196 7197

COPY entrypoint.sh /entrypoint.sh
RUN chmod a+x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
CMD [ "--sqlite-db", "/data/data.sqlite", "--sqlite-key", "password", "--cert-path", "/priv/server.crt", "--key-path", "/priv/server.key", "-log-level", "debug" ]