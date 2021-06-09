#!/usr/bin/env bash
set -ex
ALERTMANAGER_CONFIG_FILE=${ALERTMANAGER_CONFIG_FILE:-/etc/alertmanager/config.yml}
ALERTMANAGER_STORAGE_PATH=${ALERTMANAGER_STORAGE_PATH:-/alertmanager}

/bin/alertmanager --config.file="${ALERTMANAGER_CONFIG_FILE}" --storage.path="${ALERTMANAGER_STORAGE_PATH}"
