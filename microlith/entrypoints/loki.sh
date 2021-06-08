#!/usr/bin/env bash
set -e

# Set up Prometheus scraping for this target - this allows us to dynamically turn it on/off
PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/monitoring/}
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

/usr/bin/loki -config.file=/etc/loki/local-config.yaml
