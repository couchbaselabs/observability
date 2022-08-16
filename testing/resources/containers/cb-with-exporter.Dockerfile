ARG COUCHBASE_SERVER_IMAGE="couchbase/server:enterprise-6.6.3"
ARG GOLANG_VERSION=1.19

# Build the exporter
FROM golang:$GOLANG_VERSION as go_build
RUN git clone https://github.com/couchbase/couchbase-exporter.git /opt/couchbase-exporter

WORKDIR /opt/couchbase-exporter
ARG COUCHBASE_EXPORTER_VERSION=master
RUN git checkout "$COUCHBASE_EXPORTER_VERSION" &&\
    CGO_ENABLED=0 go build -o ./couchbase-exporter .

# Add to Couchbase Server
# hadolint ignore=DL3006
FROM $COUCHBASE_SERVER_IMAGE
EXPOSE 9091

COPY --chown=couchbase:couchbase --chmod=777 ./run_exporter.sh /etc/service/couchbase-exporter/run
COPY --from=go_build --chmod=775 /opt/couchbase-exporter/couchbase-exporter /opt/couchbase-exporter
RUN chown couchbase:couchbase /opt/couchbase-exporter
