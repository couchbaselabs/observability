# syntax=docker/dockerfile:1.3-labs
ARG version=enterprise-6.6.3
FROM couchbase:$version

RUN apt-get update && \
    apt-get install -y git curl

RUN wget https://golang.org/dl/go1.17.2.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.2.linux-amd64.tar.gz

ENV PATH="${PATH}:/usr/local/go/bin"
ENV GOROOT="/usr/local/go"

RUN mkdir -p /opt/couchbase-exporter && \
    git clone https://github.com/couchbase/couchbase-exporter.git /opt/couchbase-exporter

WORKDIR /opt/couchbase-exporter
RUN go build -o ./couchbase-exporter . && chmod +x couchbase-exporter && chown -R couchbase:couchbase /opt/couchbase-exporter

COPY ./run_exporter_when_up.sh /etc/service/couchbase-exporter/run

RUN chown -R couchbase:couchbase /etc/service && \
    chmod +x /etc/service/couchbase-exporter/run