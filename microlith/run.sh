#!/bin/bash
set -e

# Expose all nested config variables to make it simple to seeCLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
export CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}
export COUCHBASE_USER=${COUCHBASE_USER:-Administrator}
export COUCHBASE_PWD=${COUCHBASE_PWD:-password}
export COUCHBASE_ENDPOINT=${COUCHBASE_ENDPOINT:-http://db1:8091}
export HOST_PROC=${NODE_EXPORTER_HOST_PROC:-/node-exporter/host/proc}
export HOST_ROOTFS=${NODE_EXPORTER_HOST_ROOTFS:-/node-exporter/host/rootfs}
export HOST_SYS=${NODE_EXPORTER_HOST_SYS:-/node-exporter/host/sys}
export CUSTOM_COLLECTOR=${NODE_EXPORTER_CUSTOM_COLLECTOR:-/node-exporter/custom}

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
            # See https://github.com/hilbix/speedtests for log name pre-pending info
            echo "[ENTRYPOINT] Enabled as ${DISABLE_VAR} not set: $i"
            "$i" "$@" 2>&1 | awk '{ print "['"${EXE_NAME}"']" $0 }' &
        else
            echo "[ENTRYPOINT] Skipping non-executable: $i"
        fi
    done

    wait -n
fi