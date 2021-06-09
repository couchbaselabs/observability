#!/usr/bin/env bash
set -ex

LOKI_CONFIG_FILE=${LOKI_CONFIG_FILE:-/etc/loki/local-config.yaml}

# Set up Prometheus scraping for this target - this allows us to dynamically turn it on/off
PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"
cat > "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"/loki.json << __EOF__
[
    {
      "targets": [
        "localhost:3100"
      ],
      "labels": {
        "job": "loki",
        "container": "monitoring"
      }
    }
]
__EOF__

/usr/bin/loki -config.file="${LOKI_CONFIG_FILE}"
