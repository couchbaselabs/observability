ARG couchbase_server_image=couchbase/server:enterprise-6.6.3
ARG golang_version=1.17.2

# Build the exporter
FROM golang:$golang_version as go_build
RUN git clone https://github.com/couchbase/couchbase-exporter.git /opt/couchbase-exporter

WORKDIR /opt/couchbase-exporter
RUN CGO=0 go build -o ./couchbase-exporter .

# Add to Couchbase Server
# hadolint ignore=DL3006
FROM $couchbase_server_image

COPY --chown=couchbase:couchbase --chmod=777 ./run_exporter.sh /etc/service/couchbase-exporter/run
COPY --from=go_build --chmod=777 /opt/couchbase-exporter /opt/couchbase-exporter
RUN  chown -R couchbase:couchbase /opt/couchbase-exporter


