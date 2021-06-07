#!/usr/bin/env bash
set -ex
export CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}

# Substitute environment variables as Prometheus does not support this (actively refused to do so)
# https://www.robustperception.io/environment-substitution-with-docker
envsubst < /etc/prometheus/prometheus.yml > /etc/prometheus/prometheus-runtime.yml

/bin/prometheus --config.file=/etc/prometheus/prometheus-runtime.yml \
                --storage.tsdb.path=/prometheus \
                --web.console.libraries=/usr/share/prometheus/console_libraries \
                --web.console.templates=/usr/share/prometheus/consoles \
                --web.external-url=/prometheus/

# https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus