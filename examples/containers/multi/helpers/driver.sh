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
#   - The cbs_exp_image_name Docker image built (handled by the run.sh entrypoint)
#   - A non-zero $NUM_NODES (a default is specified in the run.sh entrypoint)

# Post-conditions:
#   - $NUM_NODES Couchbase Server/exporter containers with Couchbase Server ready
#     to receive requests, and the exporter actively exporting data
function start_new_nodes() {

    local num_nodes=$1
    local cbs_exp_image_name=$2
    local nodes_ready=() 

    echo "---- Starting $num_nodes Couchbase Server and Exporter nodes ----"

    local i=0
    for ((i; i<num_nodes; i++)); do
        docker run -d --rm --name "node$i" --hostname="node$i.local" --network=multi_shared_network \
        -p $((8091+i)):8091 "$cbs_exp_image_name" > /dev/null
        nodes_ready+=(false)
    done

    # Simple block until all nodes ready
    echo "Waiting for nodes to come up..." && sleep "$num_nodes"
    while true; do
        local j=0
        for ((j; j<num_nodes; j++)); do
            if ! ${nodes_ready[$j]}; then
                if docker exec "node$j" curl -fs localhost:8091 > /dev/null; then
                    nodes_ready[$j]=true
                    echo "Node $j ready!"
                fi
            fi
        done

        ready=true
        for b in "${nodes_ready[@]}"; do if ! $b; then ready=false; fi; done
        if $ready; then break; else echo "..." && sleep 5; fi
        # Bash does not support boolean operators on true/false and the evaluation of 0 or 1 as true/false changes 
        # depending on the context and would be much harder to understand
    done

}

#########################
# Pre-conditions:
# - The passed docker container name/ID is valid and running
# - The command passed is a valid bash command to be executed inside the container
# - $success_msg is returned by the command upon successful execution

# Post-conditions:
# - The command is executed successfully with no error code; OR
# - The command fails $retry_count times in a row and the program exits
function _docker_exec_with_retry() {

    local retry_count=5
    local retry_time=(2 4 8 15 30)

    local container=$1
    local command=$2
    local success_msg=${3:-""}

    local i=0
    for ((i; i<retry_count; i++)); do
        output=$(docker exec "$container" /usr/bin/env bash -c "$command")
        if [[ $output == *$success_msg* ]]; then
            return
        else
            echo "Command failed, waiting ${retry_time[$i]} seconds before retrying..."
            sleep "${retry_time[$i]}"
            
        fi
    done
    echo "Max retries reached while executing $command, $container failed for reason: $output"
    exit 1

}
# Pre-conditions: 
#   - $num_nodes containers running Couchbase Server (uninitialised)/exporter 

# Post-conditions: 
#   - All CBS/exporter nodes initialised and partitioned as evenly as possible into 
#     $num_clusters clusters, with a rebalance occurring after the last node is added
#   - $num_clusters nodes will be running the Data Service, the rest Index/Query, with quotas
#     specified by $data_alloc and $index_alloc
#   - Every cluster registered for monitoring with the cbmultimanager
function configure_servers() {

    local num_nodes=$1
    local num_clusters=$2
    local server_user=$3
    local server_pwd=$4
    local node_ram=$5
    local load=$6

    local data_alloc 
    local index_alloc
    # Allocate 70% of the specified RAM quota to the service (query has no quota)
    # awk used as bash does not support operations with decimals
    data_alloc=$(awk -v n="$node_ram" 'BEGIN {printf "%.0f\n", (n*0.7)}')
    index_alloc=$(awk -v n="$node_ram" 'BEGIN {printf "%.0f\n", (n*0.7)}')

    local sample_buckets=(\"travel-sample\" \"beer-sample\") # Only used if LOAD is true

    local temp_dir
    temp_dir=$(mktemp -d)
    echo '[]' > "$temp_dir"/targets.json

    echo "----- START CONFIGURING NODES -----"
    echo "Partitioning $num_nodes nodes into $num_clusters clusters..."
    echo ""

    local nodes_left=$num_nodes
    local i=0
    for ((i; i<num_clusters; i++)); do

        # Calculate the number of nodes to provision in this cluster
        local to_provision=$(( nodes_left / (num_clusters - i) )) # (Integer division, Bash does not support decimals)
        local start=$(( num_nodes - nodes_left ))
        
        # Create and initialize cluster
        local uid="node$start"
        local clust_name="Cluster $i"
        _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli cluster-init -c localhost --cluster-name=\"$clust_name\" \
            --cluster-username=\"$server_user\" --cluster-password=\"$server_pwd\" --cluster-ramsize=$data_alloc \
            --cluster-index-ramsize=$index_alloc --services=data || echo 'failed'" "SUCCESS: "
        local nodes=(\"node"$start".local:9091\")

        echo "** $clust_name created **"

        # Load sample buckets and register cluster with CBMM
        sample_buckets_json=$(IFS=, ; echo "${sample_buckets[*]}")
        _docker_exec_with_retry "$uid" "curl -fs -X POST -u \"$server_user\":\"$server_pwd\" \"http://localhost:8091/sampleBuckets/install\" \
          -d '[$sample_buckets_json]'" "[]"

        echo "- Sample buckets ${sample_buckets_json} loaded"
        
        if $load; then
            # Start cbpillowfight to simulate a non-zero load (NOT stress test)
            # Currently broken as & doesn't pass output with docker exec for some reason

            for bucket in "${sample_buckets[@]}"; do
                { 
                  _docker_exec_with_retry "$uid" "/opt/couchbase/bin/cbc-pillowfight -u \"$server_user\" -P \"$server_pwd\" \
                    -U http://localhost/$bucket -B 2 -I 100 --rate-limit 20" "Running." &
                } 1>/dev/null 2>&1
                echo "  - cbc-pillowfight started against $bucket"
            done
        fi

        local cmos_cmd="curl -fs -u $CLUSTER_MONITOR_USER:$CLUSTER_MONITOR_PWD -X POST -d \
          '{\"user\":\"$server_user\",\"password\":\"$server_pwd\", \"host\":\"http://$uid:8091\"}' \
          'http://localhost:8080/couchbase/api/v1/clusters'"
        _docker_exec_with_retry "cmos" "$cmos_cmd"

        echo "- Registered with Cluster Monitor"

        # Initialize and add the required nodes to the existing cluster
        echo ""
        echo "Adding $((to_provision)) nodes to cluster"
        echo " - node$start added"

        local j=$((start+1))
        for ((j; j<start+to_provision; j++)); do 
                local node="node$j"
                _docker_exec_with_retry $node "/opt/couchbase/bin/couchbase-cli node-init --cluster \"http://$uid:8091\" \
                   --username \"$server_user\" --password \"$server_pwd\" || echo 'failed'" "SUCCESS: "
                _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli server-add --cluster \"http://$uid:8091\" \
                  --username \"$server_user\" --password \"$server_pwd\" --server-add \"http://$node.local:8091\" \
                  --server-add-username \"$server_user\" --server-add-password \"$server_pwd\" --services index,query \
                  || echo 'failed'" "SUCCESS: "

                nodes+=(\""$node".local:9091\")
                echo " - $node added"
        done

        echo "All nodes added successfully."

        # Add cluster to CMOS' Prometheus JSON config file
        local csv, new_file
        csv=$(IFS=, ; echo "${nodes[*]}") # arr -> str: "node0.local:9091","node1.local:9091", ...
        new_file=$(jq ". |= .+ [{\"targets\":[$csv], \"labels\":{\"cluster\":\"$clust_name\"}}]" "$temp_dir"/targets.json)
        echo "$new_file" > "$temp_dir"/targets.json

        echo ""
        echo "- Nodes added to Prometheus scrape config under the cluster"

        # Rebalance newly-added nodes into the fully provisioned cluster
        if (( to_provision > 1 )); then
            _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli rebalance --cluster \"$uid\" \
              --username \"$server_user\" --password \"$server_pwd\" --no-progress-bar --no-wait || echo 'failed'" "SUCCESS: "
        fi

        echo "- Rebalance started"
        echo "Cluster configuration complete."
        echo ""

        local nodes_left=$((nodes_left - to_provision))

    done

    # Copy finished targets.json into CMOS container
    docker cp "$temp_dir"/targets.json cmos:/etc/prometheus/couchbase/custom/targets.json
}
