#!/usr/bin/env bash
set -ex

# Set up Prometheus scraping for this target - this allows us to dynamically turn it on/off
PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"
cat > "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"/fluentbit.json << __EOF__
[
    {
      "targets": [
        "localhost:2020"
      ],
      "labels": {
        "job": "fluentbit",
        "container": "monitoring",
        "__metrics_path__": "/api/v1/metrics/prometheus"
      }
    }
]
__EOF__

/fluent-bit/bin/fluent-bit -c /fluent-bit/etc/fluent-bit.conf