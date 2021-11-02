ARG couchbase_server_image=couchbase/server:enterprise-6.6.3
ARG exporter_version=master

# Build the exporter
FROM golang:1.17.2 as go_build
RUN git clone https://github.com/couchbase/couchbase-exporter.git /opt/couchbase-exporter

WORKDIR /opt/couchbase-exporter
RUN git checkout $exporter_version
RUN CGO_ENABLED=0 go build -o ./couchbase-exporter .

# Add to Couchbase Server
# hadolint ignore=DL3006
FROM $couchbase_server_image
EXPOSE 9091

COPY --chown=couchbase:couchbase --chmod=777 ./run_exporter.sh /etc/service/couchbase-exporter/run
COPY --from=go_build --chmod=775 /opt/couchbase-exporter/couchbase-exporter /opt/couchbase-exporter
RUN chown couchbase:couchbase /opt/couchbase-exporter
