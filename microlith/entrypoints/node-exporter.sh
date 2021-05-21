#!/usr/bin/env bash
set -e
HOST_PROC=${NODE_EXPORTER_HOST_PROC:-/node-exporter/host/proc}
HOST_ROOTFS=${NODE_EXPORTER_HOST_ROOTFS:-/node-exporter/host/rootfs}
HOST_SYS=${NODE_EXPORTER_HOST_SYS:-/node-exporter/host/sys}
CUSTOM_COLLECTOR=${NODE_EXPORTER_CUSTOM_COLLECTOR:-/node-exporter/custom}

ARGS="--collector.filesystem.ignored-mount-points=^/(sys|proc|dev|host|etc)($$|/)"
if [[ -d "${HOST_PROC}" ]]; then
    ARGS+=" --path.procfs=${HOST_PROC}"
fi
if [[ -d "${HOST_ROOTFS}" ]]; then
    ARGS+=" --path.rootfs=${HOST_ROOTFS}"
fi
if [[ -d "${HOST_SYS}" ]]; then
    ARGS+=" --path.sysfs=${HOST_SYS}"
fi
if [[ -d "${CUSTOM_COLLECTOR}" ]]; then
    ARGS+=" --collector.textfile.directory=${CUSTOM_COLLECTOR}"
fi
/bin/node_exporter "${ARGS}"
