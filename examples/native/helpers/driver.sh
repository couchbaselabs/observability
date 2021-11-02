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
#   - A non-zero $NUM_NODES (a default is specified in the run.sh entrypoint)

# Post-conditions:
#   - $NUM_NODES Couchbase Server/exporter containers with Couchbase Server ready
#     to receive requests, and the exporter actively exporting data
function start_new_nodes() {

    local NUM_NODES=$1
    local NODE_READY=() 

    local i=0
    for ((i; i<NUM_NODES; i++)); do
        docker run -d --rm --name "node$i" --hostname="node$i.local" --network=native_shared_network "cbs_server_exp"
        NODE_READY+=(false)
    done

    # Simple block until all nodes ready
    echo "Waiting for nodes to come up..." && sleep "$NUM_NODES"
    while true; do
        local j=0
        for ((j; j<NUM_NODES; j++)); do
            if ! ${NODE_READY[$j]}; then
                if docker exec "node$j" curl -fs localhost:8091; then
                    NODE_READY[$j]=true
                    echo "Node $j ready!"
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

    local RETRY_COUNT=5

    local CONTAINER=$1
    local COMMAND=$2
    local SUCCESS_MSG=${3:-""}

    local i=0
    for ((i; i<RETRY_COUNT; i++)); do
        output=$(docker exec "$CONTAINER" /usr/bin/env bash -c "$COMMAND")
        if [[ $output == *$SUCCESS_MSG* ]]; then
            return
        else
            sleep 2
            echo "Retrying..."
        fi
    done
    echo "Max retries reached, $CONTAINER failed"

}
# Pre-conditions: 
#   - $NUM_NODES containers running Couchbase Server (uninitialised)/exporter 

# Post-conditions: 
#   - All CBS/exporter nodes initialised and partitioned as evenly as possible into 
#     $CLUSTER_NUM clusters, with a rebalance occurring after the last node is added
#   - $CLUSTER_NUM nodes will be running the Data Service, the rest Index/Query, with quotas
#     specified by $DATA_ALLOC and $INDEX_ALLOC
#   - Every cluster registered for monitoring with the cbmultimanager
function configure_servers() {

    local NUM_NODES=$1
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

    local temp_dir
    temp_dir=$(mktemp -d)
    echo '[]' > "$temp_dir"/targets.json

    local nodes_left=$NUM_NODES
    local i=0
    for ((i; i<CLUSTER_NUM; i++)); do

        # Calculate the number of nodes to provision in this cluster
        local to_provision=$(( nodes_left / (CLUSTER_NUM - i) )) # This is always integer division, Bash does not support decimals
        local start=$(( NUM_NODES - nodes_left ))
        
        # Create and initialize cluster
        local uid="node$start"
        local clust_name="Cluster $i"
        _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli cluster-init -c localhost --cluster-name=\"$clust_name\" \
            --cluster-username=\"$SERVER_USER\" --cluster-password=\"$SERVER_PWD\" --cluster-ramsize=$DATA_ALLOC \
            --cluster-index-ramsize=$INDEX_ALLOC --services=data || echo 'failed'" "SUCCESS: "
        local nodes=(\"node"$start".local:9091\")

        # Load sample buckets and register cluster with CBMM
        _docker_exec_with_retry "$uid" "curl -fs -X POST -u \"$SERVER_USER\":\"$SERVER_PWD\" \"http://localhost:8091/sampleBuckets/install\" \
          -d '[\"travel-sample\", \"beer-sample\"]'" "[]"
        
        local cmos_cmd="curl -fs -u $CLUSTER_MONITOR_USER:$CLUSTER_MONITOR_PWD -X POST -d '{\"user\":\"$SERVER_USER\",\"password\":\"$SERVER_PWD\", \"host\":\"http://$uid:8091\"}' 'http://localhost:8080/couchbase/api/v1/clusters'"
        _docker_exec_with_retry "cmos" "$cmos_cmd"
        
        if $LOAD; then
            # Start cbpillowfight to simulate a non-zero load (NOT stress test)
            # Currently broken as & doesn't pass output with docker exec for some reason
            local sample_buckets=("travel-sample" "beer-sample")

            for bucket in "${sample_buckets[@]}"; do
                #get_url="http://localhost:8091/pools/default/buckets/$bucket"
                # Attempt to GET the bucket - when this returns status 200 pillowfight starts 
                #d_cmd="if (curl -fs -X POST -u \"$SERVER_USER\":\"$SERVER_PWD\" $get_url); then echo \"SUCCESS\" \
                _docker_exec_with_retry "$uid" "/opt/couchbase/bin/cbc-pillowfight -u \"$SERVER_USER\" -P \"$SERVER_PWD\" -U http://localhost/$bucket -B 100 -I 1000 --rate-limit 100" "Running..." > /dev/null & 
            done
        fi

        # Initialize and add the required nodes to the existing cluster
        local j=$((start+1))
        for ((j; j<start+to_provision; j++)); do 
                local node="node$j"
                _docker_exec_with_retry $node "/opt/couchbase/bin/couchbase-cli node-init --cluster \"http://$uid:8091\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\" || echo 'failed'" "SUCCESS: "
                _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli server-add --cluster \"http://$uid:8091\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\" \
                    --server-add \"http://$node.local:8091\" --server-add-username \"$SERVER_USER\" --server-add-password \"$SERVER_PWD\" --services index,query || echo 'failed'" "SUCCESS: "

                nodes+=(\""$node".local:8091\")
        done

        # Rebalance newly-added nodes into the fully provisioned cluster
        if (( to_provision > 1 )); then
            _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli rebalance --cluster \"$uid\" --username \"$SERVER_USER\" --password \"$SERVER_PWD\" \
            --no-progress-bar --no-wait || echo 'failed'" "SUCCESS: "
        fi

        local nodes_left=$((nodes_left - to_provision))

        # Add cluster to CMOS' Prometheus JSON config file
        bar=$(IFS=, ; echo "${nodes[*]}") # arr -> str: "node0.local:9091","node1.local:9091", ...
        new_file=$(jq ". |= .+ [{\"targets\":[$bar], \"cluster\":\"$clust_name\"}]" "$temp_dir"/targets.json)
        echo "$new_file" > "$temp_dir"/targets.json

        #
        cat "$temp_dir"/targets.json
        #

    done

    docker cp "$temp_dir"/targets.json cmos:/etc/prometheus/couchbase/custom/targets.json
}
