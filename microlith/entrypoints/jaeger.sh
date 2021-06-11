#!/usr/bin/env bash
set -ex

JAEGER_URL_SUBPATH=${JAEGER_URL_SUBPATH-/jaeger}
JAEGER_CONFIG_FILE=${JAEGER_CONFIG_FILE:-/etc/jaeger/config.json}
SPAN_STORAGE_TYPE=${SPAN_STORAGE_TYPE:-memory}

# Set up Prometheus scraping for this target - this allows us to dynamically turn it on/off
PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"
cat > "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"/jaeger.json << __EOF__
[
    {
      "targets": [
        "localhost:14269"
      ],
      "labels": {
        "job": "jaeger",
        "container": "monitoring"
      }
    }
]
__EOF__

/go/bin/all-in-one-linux --query.base-path="${JAEGER_URL_SUBPATH}" \
                         --query.ui-config="${JAEGER_CONFIG_FILE}" \
                         --admin.http.host-port ":14269"
