FROM golang:1.17-alpine AS builder

COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

COPY . /app/

RUN CGO_ENABLED=0 GOOS=linux go build -o /cmoscfg ./cmd/cmoscfg

# We require various GCC stuff for tests
FROM golang:1.17 AS unit-test
COPY . /app/
WORKDIR /app
RUN go mod download && \
    go test -trimpath -timeout 120s -v ./...

FROM alpine:3.14

# Use latest rather than pin
# hadolint ignore=DL3018
RUN apk add --upgrade --no-cache ca-certificates bash openssl

# Ensure we include licensing information
COPY ./LICENSE /licenses/cmoscfg.txt

COPY ./entrypoint.sh /entrypoint.sh

COPY --from=builder /cmoscfg /bin/cmoscfg

# Provide a configurable user to run as and for ownership
ARG CB_UID="8453"
ARG CB_GID="8453"

RUN addgroup -g $CB_GID -S couchbase && \
    adduser -u $CB_UID -S couchbase -G couchbase

# Ensure we have everything we need set up with correct permissions
RUN chmod a+x /bin/cmoscfg && \
    chmod a+x /entrypoint.sh && \
    chmod -R a+r /licenses

EXPOSE 7194

ENTRYPOINT ["/entrypoint.sh"]
