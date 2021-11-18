#!/bin/bash
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
set -eu

function startServer() {
    # Clean up old logs and make sure we have directories we need
    rm -rf /opt/couchbase/var/lib/couchbase/logs/*
    mkdir -p /opt/couchbase/var/lib/couchbase/{config,data,stats,logs}

    echo "Run up Couchbase Server"
    # The actual command to run CB server based on the server image entrypoint.
    /opt/couchbase/bin/couchbase-server -- -kernel global_enable_tracing false -noinput &

    echo "Wait for it to be ready"
    until curl -sS -w 200 http://127.0.0.1:8091/ui/index.html &> /dev/null; do
        echo "Not ready, waiting to recheck"
        sleep 2
    done
}

function configureCluster() {
    echo "Configuring cluster"
    couchbase-cli cluster-init -c 127.0.0.1 \
        --cluster-username Administrator \
        --cluster-password password \
        --services data,index,query,fts,analytics \
        --cluster-ramsize 1024 \
        --cluster-index-ramsize 512 \
        --cluster-eventing-ramsize 512 \
        --cluster-fts-ramsize 512 \
        --cluster-analytics-ramsize 1024 \
        --cluster-fts-ramsize 512 \
        --cluster-name "CMOS container test" \
        --index-storage-setting default
}

function enableAuditing() {
    echo "Enable audit logging"
    couchbase-cli setting-audit -c 127.0.0.1 \
        --username Administrator \
        --password password \
        --set \
        --audit-enabled 1
}

function createBucket() {
    echo "Creating bucket"
    couchbase-cli bucket-create -c 127.0.0.1 \
        --username Administrator \
        --password password \
        --bucket testBucket \
        --bucket-type couchbase \
        --bucket-ramsize 1024 \
        --max-ttl 500000000 \
        --durability-min-level persistToMajority \
        --enable-flush 0
}

function waitForStartup() {
    echo "Waiting for startup completion"
    local ATTEMPTS=0
    # Wait for startup - no great way for this
    until curl -u "Administrator:password" http://127.0.0.1:8091/pools/default &> /dev/null; do
        # Prevent an infinite loop - at 2 seconds per go this is 5 minutes
        if [ $ATTEMPTS -gt "150" ]; then
            echo "Unable to start up Couchbase Server"
            exit 1
        fi
        ATTEMPTS=$((ATTEMPTS+1))
        echo "Not running, waiting to recheck"
        sleep 2
    done
    echo "Running"
}

function enforceReadAccessToLogs() {
    # Ensure everyone can read the logs as new ones are created
    until ! chmod -R a+r /opt/couchbase/var/lib/couchbase/logs/; do
        sleep 10
    done
}

echo "Overriding CBS entrypoint as this gives us declarative control"
startServer

if configureCluster ; then
    enableAuditing
    waitForStartup
    createBucket
else
    echo "Cluster configuration failed, assuming already configured"
    waitForStartup
fi

echo "Set up complete"
enforceReadAccessToLogs

echo "Exiting"
