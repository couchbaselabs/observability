FROM golang:1.16 as healthcheck-builder

RUN apt-get update && \
    apt-get install -y sqlcipher libssl-dev openssl openssh-client git && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /src/github.com/couchbaselabs/cbmultimanager/ /bin

# Support cloning the private repo
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN --mount=type=ssh git clone --depth 1 git@github.com:couchbaselabs/cbmultimanager.git /src/github.com/couchbaselabs/cbmultimanager/

WORKDIR /src/github.com/couchbaselabs/cbmultimanager

# Statically build it to allow reuse on Alpine Linux
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o /bin/cbeventlog ./cmd/cbeventlog

FROM couchbase/fluent-bit:1.0.3
COPY --from=healthcheck-builder /bin/cbeventlog /fluent-bit/bin/cbeventlog
COPY entrypoint.sh /entrypoint.sh
COPY fluent-bit.conf.server /fluent-bit/config/fluent-bit.conf

RUN chmod a+x /fluent-bit/bin/cbeventlog /entrypoint.sh
EXPOSE 2020
CMD [ "/entrypoint.sh" ]
