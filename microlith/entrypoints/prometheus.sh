#!/usr/bin/env bash
set -ex
export CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}

# To customise the Prometheus configuration used, set these values at launch
PROMETHEUS_CONFIG_FILE=${PROMETHEUS_CONFIG_FILE:-/etc/prometheus/prometheus-runtime.yml}
PROMETHEUS_CONFIG_TEMPLATE_FILE=${PROMETHEUS_CONFIG_TEMPLATE_FILE:-/etc/prometheus/prometheus-template.yml}
PROMETHEUS_URL_SUBPATH=${PROMETHEUS_URL_SUBPATH-/prometheus/}
PROMETHEUS_STORAGE_PATH=${PROMETHEUS_STORAGE_PATH-/prometheus}

# Substitute environment variables as Prometheus does not support this (actively refused to do so)
# https://www.robustperception.io/environment-substitution-with-docker
if [[ -f "${PROMETHEUS_CONFIG_TEMPLATE_FILE}" ]] ; then
  envsubst < "${PROMETHEUS_CONFIG_TEMPLATE_FILE}" > "${PROMETHEUS_CONFIG_FILE}"
fi

# Add in metric support for pushgateway if enabled - it runs its own binary separately
if [[ -v "DISABLE_PUSHGATEWAY" ]]; then
  echo "DISABLE_PUSHGATEWAY set so no endpoint to create"
else
  echo "Creating pushgateway metric endpoint"
  PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
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

# From: https://prometheus.io/docs/prometheus/latest/configuration/configuration/
# A configuration reload is triggered by sending a SIGHUP to the Prometheus process or
# sending a HTTP POST request to the /-/reload endpoint.

/bin/prometheus --config.file="${PROMETHEUS_CONFIG_FILE}" \
                --storage.tsdb.path="${PROMETHEUS_STORAGE_PATH}" \
                --web.console.libraries=/usr/share/prometheus/console_libraries \
                --web.console.templates=/usr/share/prometheus/consoles \
                --web.external-url="${PROMETHEUS_URL_SUBPATH}" \
                --web.enable-lifecycle

# https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus