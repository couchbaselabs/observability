#!/usr/bin/env bash
set -ex
export CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}

# Substitute environment variables as Prometheus does not support this (actively refused to do so)
# https://www.robustperception.io/environment-substitution-with-docker
envsubst < /etc/prometheus/prometheus.yml > /etc/prometheus/prometheus-runtime.yml

# Add in metric support for pushgateway if enabled - it runs its own binary separately
if [[ -v "DISABLE_PUSHGATEWAY" ]]; then
    PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/monitoring/}
    mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"
    cat > "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"/pushgateway.json << __EOF__
[
    {
      "targets": [
        "localhost:9091"
      ],
      "labels": {
        "job": "pushgateway",
        "container": "monitoring"
      }
    }
]
__EOF__

fi

/bin/prometheus --config.file=/etc/prometheus/prometheus-runtime.yml \
                --storage.tsdb.path=/prometheus \
                --web.console.libraries=/usr/share/prometheus/console_libraries \
                --web.console.templates=/usr/share/prometheus/consoles \
                --web.external-url=/prometheus/

# https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus