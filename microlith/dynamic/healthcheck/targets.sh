#!/usr/bin/env bash
set -eux
# Simple file containing cluster id and credentials to register with healthcheck.
# We need to feed it a matched triplet for the JSON above, so endpoint, username and password for the couchbase cluster.
# Customise or copy as necessary and all shell scripts in this directory will be periodically run.
CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}
COUCHBASE_USER=${COUCHBASE_USER:-Administrator}
COUCHBASE_PWD=${COUCHBASE_PWD:-password}
COUCHBASE_ENDPOINT=${COUCHBASE_ENDPOINT:-http://db1:8091}
curl -u "${CLUSTER_MONITOR_USER}:${CLUSTER_MONITOR_PWD}" -X POST -d '{ "user": "'"${COUCHBASE_USER}"'", "password": "'"${COUCHBASE_PWD}"'", "host": "'"${COUCHBASE_ENDPOINT}"'" }' "${CLUSTER_MONITOR_ENDPOINT}/api/v1/clusters"
# Otherwise can run the command separately.