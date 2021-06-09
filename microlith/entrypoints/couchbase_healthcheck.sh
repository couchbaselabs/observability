#!/usr/bin/env bash
set -ex
CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}
COUCHBASE_USER=${COUCHBASE_USER:-Administrator}
COUCHBASE_PWD=${COUCHBASE_PWD:-password}
COUCHBASE_ENDPOINT=${COUCHBASE_ENDPOINT:-http://db1:8091}

/bin/cbmultimanager --sqlite-db /data/data.sqlite --sqlite-key password --cert-path /priv/server.crt --key-path /priv/server.key -log-level debug &
set +x

# From: https://github.com/couchbaselabs/cbmultimanager/wiki/Basic-REST-API-usage
# Must be in JSON format

# Configure access to cluster monitor
until curl -X POST -H "Content-Type: application/json" -d '{"user": "'"${CLUSTER_MONITOR_USER}"'", "password": "'"${CLUSTER_MONITOR_PWD}"'" }' "${CLUSTER_MONITOR_ENDPOINT}/api/v1/self" ; do
    sleep 5
done

# Configure clusters to monitor - unfortunately you have to wait for the couchbase endpoint to come up
sleep 60

# Issue with using --log-path: https://github.com/couchbaselabs/cbmultimanager/issues/39
mkdir -p /opt/couchbase/var/lib/couchbase/logs/db1 && cd /opt/couchbase/var/lib/couchbase/logs/db1/

while true; do
    # Periodically we scan for stuff to do, e.g. register new clusters
    if [[ -d /etc/healthcheck/ ]]; then
        for SCRIPT in /etc/healthcheck/*.sh; do
            [[ ! -f $SCRIPT ]] && continue
            echo "Using dynamic script: $SCRIPT"
            /bin/bash "$SCRIPT"
        done
    fi

    # Run the event log generator
    # TODO: replace with usage of replicated logs or move to fluent bit itself and send to loki: https://github.com/couchbaselabs/cbmultimanager/issues/33
    sleep 30
    /bin/cbeventlog node --username "${COUCHBASE_USER}" --password "${COUCHBASE_PWD}" --node db1 --node-name db1
done

wait -n
