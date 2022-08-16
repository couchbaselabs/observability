FROM golang:1.19 AS codebase
# Cache Go dependencies as separate layer
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download
# Now add rest of source
COPY . /app/

FROM codebase AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -a -o /cmoscfg -tags netgo ./cmd/cmoscfg

FROM codebase AS unit-test
# We require various GCC stuff for tests
RUN go test -trimpath -timeout 120s -v ./...

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

# Ensure we have everything we need set up with correct permissions
RUN addgroup -g $CB_GID -S couchbase && \
    adduser -u $CB_UID -S couchbase -G couchbase && \
    chmod a+x /bin/cmoscfg && \
    chmod a+x /entrypoint.sh && \
    chmod -R a+r /licenses

EXPOSE 7194
ENTRYPOINT ["/entrypoint.sh"]
