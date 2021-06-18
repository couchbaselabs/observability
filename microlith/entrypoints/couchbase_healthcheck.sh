#!/usr/bin/env bash
# Copyright 2021 Couchbase, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file  except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the  License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex
CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
CLUSTER_MONITOR_ENDPOINT=${CLUSTER_MONITOR_ENDPOINT:-http://localhost:7196}
COUCHBASE_USER=${COUCHBASE_USER:-Administrator}
COUCHBASE_PWD=${COUCHBASE_PWD:-password}
COUCHBASE_ENDPOINT=${COUCHBASE_ENDPOINT:-http://db1:8091}

if [[ -x "/bin/cbmultimanager" ]]; then
    /bin/cbmultimanager --sqlite-db /data/data.sqlite --sqlite-key password --cert-path /priv/server.crt --key-path /priv/server.key -log-level debug &
else
    echo "No healthchecker to run"
fi
set +x

# From: https://github.com/couchbaselabs/cbmultimanager/wiki/Basic-REST-API-usage
# Must be in JSON format

# Configure access to cluster monitor
until curl -X POST -H "Content-Type: application/json" -d '{"user": "'"${CLUSTER_MONITOR_USER}"'", "password": "'"${CLUSTER_MONITOR_PWD}"'" }' "${CLUSTER_MONITOR_ENDPOINT}/api/v1/self" ; do
    sleep 5
done

# Configure clusters to monitor - unfortunately you have to wait for the couchbase endpoint to come up
sleep 60

mkdir -p /opt/couchbase/var/lib/couchbase/logs/db1

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
    sleep 60
    if [[ -x "/bin/cbeventlog" ]]; then
        /bin/cbeventlog node --username "${COUCHBASE_USER}" --password "${COUCHBASE_PWD}" --node db1 --node-name db1 --log-path /opt/couchbase/var/lib/couchbase/logs/db1 --output-path /opt/couchbase/var/lib/couchbase/logs/db1/
    else
        echo "No event log generator to run"
    fi
done

wait -n
