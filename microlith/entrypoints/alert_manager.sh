#!/usr/bin/env bash
set -ex
ALERTMANAGER_CONFIG_FILE=${ALERTMANAGER_CONFIG_FILE:-/etc/alertmanager/config.yml}
ALERTMANAGER_STORAGE_PATH=${ALERTMANAGER_STORAGE_PATH:-/alertmanager}
ALERTMANAGER_URL_SUBPATH=${ALERTMANAGER_URL_SUBPATH-/alertmanager/}

/bin/alertmanager --config.file="${ALERTMANAGER_CONFIG_FILE}" --storage.path="${ALERTMANAGER_STORAGE_PATH}"  --web.route-prefix="${ALERTMANAGER_URL_SUBPATH}"
