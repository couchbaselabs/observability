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

#########################
# Pre-conditions:
#   - The cbs_server_exp Docker image built (handled by the run.sh entrypoint)
#   - A non-zero $NODE_NUM (a default is specified in the run.sh entrypoint)

# Post-conditions:
#   - $NODE_NUM Couchbase Server/exporter containers with Couchbase Server ready
#     to receive requests, and the exporter actively exporting data
function start_new_nodes() {

    local NODE_NUM=$1
    local NODE_READY=() 

    for ((i=0; i<NODE_NUM; i++)); do
        docker run -d --name "node$i" --hostname="node$i.local" --network=native_shared_network "cbs_server_exp"
        NODE_READY+=(false)
    done

    # Simple block until all nodes ready
    echo "Waiting for nodes to come up..." && sleep "$NODE_NUM"
    while true; do
        for ((i=0; i<NODE_NUM; i++)); do
            if ! ${NODE_READY[$i]}; then
                if docker exec "node$i" curl -fs localhost:8091; then
                    NODE_READY[$i]=true
                    echo "Node $i ready!"
                fi
            fi
        done

        ready=true
        for b in "${NODE_READY[@]}"; do if ! $b; then ready=false; fi; done
        if $ready; then break; else echo "..." && sleep 5; fi
    done

}

#########################

function _docker_exec_with_retry() {

    local CONTAINER=$1
    local COMMAND=$2
    local WAIT_ERROR=${3:-"ERROR: Internal server error"}

    output=$WAIT_ERROR
    until [[ $output != *$WAIT_ERROR* ]]; do
        output=$(docker exec "$CONTAINER" /usr/bin/env bash -c "$COMMAND")
        sleep 2
    done

}
# Pre-conditions: 
#   - $NODE_NUM containers running Couchbase Server (uninitialised)/exporter 

# Post-conditions: 
#   - All CBS/exporter nodes initialised and partitioned as evenly as possible into 
#     $CLUSTER_NUM clusters, with a rebalance occurring after the last node is added
#   - $CLUSTER_NUM nodes will be running the Data Service, the rest Index/Query, with quotas
#     specified by $DATA_ALLOC and $INDEX_ALLOC
#   - Every cluster registered for monitoring with the cbmultimanager
function configure_servers() {

    local NODE_NUM=$1
    local CLUSTER_NUM=$2
    local SERVER_USER=$3
    local SERVER_PWD=$4
    local NODE_RAM=$5
    local LOAD=$6

    local DATA_ALLOC 
    local INDEX_ALLOC
    # Allocate 70% of the specified RAM quota to the service (query has no quota)
    DATA_ALLOC=$(awk -v n="$NODE_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')
    INDEX_ALLOC=$(awk -v n="$NODE_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')

    local nodes_left=$NODE_NUM

    for ((i=0; i<CLUSTER_NUM; i++)); do

        # Calculate the number of nodes to provision in this cluster
        local to_provision=$(( nodes_left / (CLUSTER_NUM - i) )) # This is always integer division, Bash does not support decimals
        local start=$(( NODE_NUM - nodes_left ))

        # Create and initialize cluster
        local uid="node$start"
        local clust_name="Cluster $i"
        _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli cluster-init -c localhost --cluster-name=\"$clust_name\" \
            --cluster-username=\"$SERVER_USER\" --cluster-password=\"$SERVER_PWD\" --cluster-ramsize=$DATA_ALLOC \
            --cluster-index-ramsize=$INDEX_ALLOC --services=data"

        # Load sample buckets and register cluster with CBMM
        _docker_exec_with_retry "$uid" "curl -X POST -u \"$SERVER_USER\":\"$SERVER_PWD\" \"http://localhost:8091/sampleBuckets/install\" -d '[\"travel-sample\", \"beer-sample\"]'"
        
        local cmos_cmd="curl -u $CLUSTER_MONITOR_USER:$CLUSTER_MONITOR_PWD -X POST -d '{\"user\":\"$SERVER_USER\",\"password\":\"$SERVER_PWD\", \
          \"host\":\"http://$uid:8091\"}' 'http://localhost:8080/couchbase/api/v1/clusters'"
        _docker_exec_with_retry "cmos" "$cmos_cmd"
        
        if $LOAD; then
            # Start cbpillowfight to simulate a non-zero load (NOT stress test)
            # Currently broken as & doesn't pass output with docker exec for some reason
            local sample_buckets=("travel-sample" "beer-sample")

            for bucket in "${sample_buckets[@]}"; do
                get_url="http://localhost:8091/pools/default/buckets/$bucket"
                # Attempt to GET the bucket - when this returns status 200 pillowfight starts 
                d_cmd="if ! (curl -X POST $get_url); then echo \"NOT_READY\" else /opt/couchbase/bin/cbc-pillowfight -u \"$SERVER_USER\" -P \"$SERVER_PWD\" \
                  -U http://localhost/$bucket -B 100 -I 1000 --rate-limit 100 & fi"
                
                _docker_exec_with_retry "$uid" "$d_cmd" "NOT_READY"
            done
        fi

        # Initialize and add the required nodes to the existing cluster
        for ((j=start+1; j<start+to_provision; j++)); do 
                local node="node$j"
                _docker_exec_with_retry $node "/opt/couchbase/bin/couchbase-cli node-init --cluster \"http://$uid:8091\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\""
                _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli server-add --cluster \"http://$uid:8091\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\" \
                    --server-add \"http://$node.local:8091\" --server-add-username \"$SERVER_USER\" --server-add-password \"$SERVER_PWD\" --services index,query"
        done

        # Rebalance newly-added nodes into the fully provisioned cluster
        if (( to_provision > 1 )); then
            _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli rebalance --cluster \"$uid\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\" \
            --no-progress-bar --no-wait"
        fi

        local nodes_left=$((nodes_left - to_provision))

    done

}
