#!/bin/bash
#
# Copyright (C) 2021 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#
set -eu

# These should be set from Secrets mounted in for production usage
CB_MULTI_ADMIN_USER=${CB_MULTI_ADMIN_USER:-admin}
CB_MULTI_ADMIN_PASSWORD=${CB_MULTI_ADMIN_PASSWORD:-password}
CB_MULTI_SQLITE_PASSWORD=${CB_MULTI_SQLITE_PASSWORD:-password}

# Certificates and persistent stores if need to be relocated
CB_MULTI_SQLITE_PATH=${CB_MULTI_SQLITE_PATH:-/data/data.sqlite}
CB_MULTI_CERT_PATH=${CB_MULTI_CERT_PATH:-/priv/server.crt}
CB_MULTI_KEY_PATH=${CB_MULTI_KEY_PATH:-/priv/server.key}

# Additional configuration as required, either via env vars or config map
CB_MULTI_UI_PATH=${CB_MULTI_UI_PATH:-/ui}
CB_MULTI_LOG_LEVEL=${CB_MULTI_LOG_LEVEL:-debug}
CB_MULTI_BIN=${CB_MULTI_BIN:-/bin/cbmultimanager}
# These turn on/off the various APIs available
CB_MULTI_ENABLE_ADMIN_API=${CB_MULTI_ENABLE_ADMIN_API:-true}
CB_MULTI_ENABLE_CLUSTER_API=${CB_MULTI_ENABLE_CLUSTER_API:-true}
CB_MULTI_ENABLE_EXTENDED_API=${CB_MULTI_ENABLE_EXTENDED_API:-true}

# These are only necessary if you're using Prometheus discovery
CB_MULTI_PROMETHEUS_URL=${CB_MULTI_PROMETHEUS_URL:-""}
CB_MULTI_PROMETHEUS_LABEL_SELECTOR=${CB_MULTI_PROMETHEUS_LABEL_SELECTOR:-""}
CB_MULTI_COUCHBASE_USER=${CB_MULTI_COUCHBASE_USER:-""}
CB_MULTI_COUCHBASE_PASSWORD=${CB_MULTI_COUCHBASE_PASSWORD:-""}

# These are only necessary if using Alertmanager
CB_MULTI_ALERTMANAGER_URLS=${CB_MULTI_ALERTMANAGER_URLS:-""}
CB_MULTI_ALERTMANAGER_RESEND_DELAY=${CB_MULTI_ALERTMANAGER_RESEND_DELAY:-1m}
CB_MULTI_ALERTMANAGER_BASE_LABELS=${CB_MULTI_ALERTMANAGER_BASE_LABELS:-""}

echo "By using this software you accept the Couchbase license agreement which can be found in /licenses/couchbase.txt or at https://www.couchbase.com/LA03012021"
echo "To see all licenses just run with a custom command to list them all, e.g. docker run ... cat /licenses/*"

if [[ $# -gt 0 ]]; then
    echo "Running custom: $*"
    exec "$@"
else
    if [[ -x "${CB_MULTI_BIN}" ]]; then
        # Making all parameters explicit so people can see how to configure the CLI.
        exec "${CB_MULTI_BIN}"  --sqlite-key "${CB_MULTI_SQLITE_PASSWORD}" \
                                --sqlite-db "${CB_MULTI_SQLITE_PATH}" \
                                --cert-path "${CB_MULTI_CERT_PATH}" \
                                --key-path "${CB_MULTI_KEY_PATH}" \
                                --log-level "${CB_MULTI_LOG_LEVEL}" \
                                --admin-user "${CB_MULTI_ADMIN_USER}" \
                                --admin-password "${CB_MULTI_ADMIN_PASSWORD}" \
                                --enable-admin-api="${CB_MULTI_ENABLE_ADMIN_API}" \
                                --enable-cluster-management-api="${CB_MULTI_ENABLE_CLUSTER_API}" \
                                --enable-extended-api="${CB_MULTI_ENABLE_EXTENDED_API}" \
                                --prometheus-url "${CB_MULTI_PROMETHEUS_URL}" \
                                --prometheus-label-selector "${CB_MULTI_PROMETHEUS_LABEL_SELECTOR}" \
                                --alertmanager-base-labels "${CB_MULTI_ALERTMANAGER_BASE_LABELS}" \
                                --couchbase-user "${CB_MULTI_COUCHBASE_USER}" \
                                --couchbase-password "${CB_MULTI_COUCHBASE_PASSWORD}" \
                                --alertmanager-urls "${CB_MULTI_ALERTMANAGER_URLS}" \
                                --alertmanager-resend-delay "${CB_MULTI_ALERTMANAGER_RESEND_DELAY}"
    else
        echo "ERROR: No executable to run: CB_MULTI_BIN=${CB_MULTI_BIN}"
    fi
fi