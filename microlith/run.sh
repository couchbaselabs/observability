#!/bin/bash
set -e

# Expose all nested config variables to make it simple to seeCLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
export CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}
export COUCHBASE_USER=${COUCHBASE_USER:-Administrator}
export COUCHBASE_PWD=${COUCHBASE_PWD:-password}
export COUCHBASE_ENDPOINT=${COUCHBASE_ENDPOINT:-http://db1:8091}
export PROMETHEUS_CONFIG_FILE=${PROMETHEUS_CONFIG_FILE:-/etc/prometheus/prometheus-runtime.yml}
export PROMETHEUS_CONFIG_TEMPLATE_FILE=${PROMETHEUS_CONFIG_TEMPLATE_FILE:-/etc/prometheus/prometheus-template.yml}
export PROMETHEUS_SUBPATH=${PROMETHEUS_SUBPATH-/prometheus/}

# Clean up dynamic targets generated
export PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/monitoring/}
rm -rf "${PROMETHEUS_DYNAMIC_INTERNAL_DIR:?}"/
mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"

# Support passing in custom command to run, e.g. bash
if [[ $# -gt 0 ]]; then
    echo "[ENTRYPOINT] Running custom: $*"
    exec "$@"
else
    for i in /entrypoints/*; do
        EXE_NAME=${i##/entrypoints/}
        UPPERCASE=${EXE_NAME^^}
        DISABLE_VAR=DISABLE_${UPPERCASE%%.*}
        # Set DISABLE_XXX to skip running
        if [[ -v "${DISABLE_VAR}" ]]; then
            echo "[ENTRYPOINT] Disabled as ${DISABLE_VAR} set (value ignored): $i"
        elif [[ -x "$i" ]]; then
            # For performance or other reasons we may just want to log to discrete files, watch out for size
            if [[ -v "ENABLE_LOG_TO_FILE" ]]; then
                echo "[ENTRYPOINT] Running: $i ==> /logs/${EXE_NAME}.log"
                "$i" "$@" &> /logs/"${EXE_NAME}".log &
            else
                echo "[ENTRYPOINT] Running: $i"
                # See https://github.com/hilbix/speedtests for log name pre-pending info
                "$i" "$@" 2>&1 | awk '{ print "['"${EXE_NAME}"']" $0 }' &
            fi
        else
            echo "[ENTRYPOINT] Skipping non-executable: $i"
        fi
    done

    wait -n
fi