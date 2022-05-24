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
    local nodes_ready=()

    echo "---- Starting $num_nodes nodes ----"

    local i=0
    for ((i; i<num_nodes; i++)); do
        docker run -d --name "node$i" --hostname="node$i.local" --network=multi_shared_network \
        -p $((8091+i)):8091 "cbs_server_exp" > /dev/null
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
        for b in "${nodes_ready[@]}"; do
            if ! $b; then
                ready=false
            fi
        done
        if $ready; then
            break
        else
            echo "..."
            sleep 5
        fi
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

    local retry_count=10
    local retry_time=(5 10 10 10 10 10 10 10 10 10)

    local container=$1
    local command=$2
    local success_msg=${3:-""} # Couchbase REST curl commands return nothing upon success

    local i=0
    for ((i; i<retry_count; i++)); do
        output=$(docker exec "$container" /usr/bin/env bash -c "$command")
        if [[ $output == *$success_msg* ]]; then
            return
        else
            echo "Couchbase Server is not ready, waiting ${retry_time[$i]} seconds before retrying..."
            sleep "${retry_time[$i]}"

        fi
    done
    echo "Max retries reached while executing $command, $container failed for reason: $output"
    exit 1

}

function _add_nodes_to_cluster() {

    local start=$1
    local to_provision=$2
    local server_user=$3
    local server_pwd=$4

    local j=$((start+1))
    for ((j; j<start+to_provision; j++)); do

        local node="node$j"
        _docker_exec_with_retry $node "/opt/couchbase/bin/couchbase-cli node-init --cluster \"http://localhost:8091\" \
            --username \"$server_user\" --password \"$server_pwd\" || echo 'failed'" "SUCCESS: "
        _docker_exec_with_retry "$node" "/opt/couchbase/bin/couchbase-cli server-add --cluster \"http://node$start.local:8091\" \
            --username \"$server_user\" --password \"$server_pwd\" --server-add \"http://$node.local:8091\" \
            --server-add-username \"$server_user\" --server-add-password \"$server_pwd\" --services data,index,query,fts,eventing \
            || echo 'failed'" "SUCCESS: "

        echo " - $node added"
    done

}

function _load_sample_buckets() {

    local uid=$1
    local load=$2
    local server_user=$3
    local server_pwd=$4
    shift 4 # Bash passes each element in the array as another $i, so we must shift previous arguments so only array args are left
    local sample_buckets=("$@") # Then we are able to capture all remaining args in a new array

    sample_buckets_json=$(IFS=, ; echo "${sample_buckets[*]}")
    _docker_exec_with_retry "$uid" "curl -s -X POST -u \"$server_user\":\"$server_pwd\" \"http://localhost:8091/sampleBuckets/install\" \
        -d '[$sample_buckets_json]' || echo 'failed'" "[]"

    echo "- Sample buckets ${sample_buckets_json} loading in the background..."

    if $load; then # Start cbpillowfight and n1qlback to simulate a non-zero load (NOT stress test)
        sleep 10

        for bucket in "${sample_buckets[@]}"; do
            # Block until bucket is ready
            _docker_exec_with_retry "$uid" "curl -s -u \"$server_user\":\"$server_pwd\" http://localhost:8091/pools/default/buckets/$bucket \
                || echo 'failed'" "{"
            {
                _docker_exec_with_retry "$uid" "/opt/couchbase/bin/cbc-pillowfight -u \"$server_user\" -P \"$server_pwd\" \
                -U http://localhost/$bucket -B 2 -I 100 --rate-limit 20 || echo 'failed'" "Running." &
            } 1>/dev/null 2>&1
            echo "  - cbc-pillowfight started against $bucket"

            if [ "$bucket" == '"travel-sample"' ]; then
                {
                    docker cp "$SCRIPT_DIR"/helpers/demo-queries.txt "$uid":./
                    _docker_exec_with_retry "$uid" "/opt/couchbase/bin/cbc-n1qlback -U http://localhost/$bucket \
                    -u \"$server_user\" -P \"$server_pwd\" -t 1 -f ./demo-queries.txt || echo 'failed'" "Loaded" &
                } 1>/dev/null 2>&1
                echo "  - Demo queries started against travel-sample"
            fi
        done
    fi

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
    local oss_flag=$7

    local data_alloc
    local index_alloc
    # Allocate 70% of the specified RAM quota to the service (query has no quota)
    # awk used as bash does not support operations with decimals
    data_alloc=$(awk -v n="$node_ram" 'BEGIN {printf "%.0f\n", (n*0.7)}')
    index_alloc=$(awk -v n="$node_ram" 'BEGIN {printf "%.0f\n", (n*0.7)}')

    local sample_buckets=(\"travel-sample\" \"beer-sample\")

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
        _docker_exec_with_retry $uid "/opt/couchbase/bin/couchbase-cli node-init --cluster localhost \
            --node-init-hostname=$uid.local --username \"$server_user\" --password \"$server_pwd\" || echo 'failed'" "SUCCESS: "
        _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli cluster-init -c localhost --cluster-name=\"$clust_name\" \
            --cluster-username=\"$server_user\" --cluster-password=\"$server_pwd\" --cluster-ramsize=$data_alloc \
            --cluster-index-ramsize=$index_alloc --services=data || echo 'failed'" "SUCCESS: "

        echo "** $clust_name created **"

        # Register cluster with CBMM if non-OSS build
        if ! $oss_flag; then
            local cmos_cmd="curl -s -u $CLUSTER_MONITOR_USER:$CLUSTER_MONITOR_PWD -X POST -d \
              '{\"user\":\"$server_user\",\"password\":\"$server_pwd\", \"host\":\"http://$uid.local:8091\"}' \
            'http://localhost:8080/couchbase/api/v1/clusters'"
            _docker_exec_with_retry "cmos" "$cmos_cmd || echo 'failed'" ""

            echo "- Registered with Cluster Monitor"
        else
            echo "- Skipped registering with Cluster Monitor (oss-build)."
        fi

        # Initialize and add the required nodes to the existing cluster
        echo ""
        echo "Adding $((to_provision)) nodes to cluster"
        echo " - node$start added"

        _add_nodes_to_cluster "$start" "$to_provision" "$server_user" "$server_pwd"

        echo "All nodes added successfully."

        # Add cluster to CMOS' Prometheus using the config service. From its OpenAPI spec the management port
        # defaults to 8091, and depending upon the Server version looks for Prometheus metrics on either
        # 8091 (Couchbase Server 7.0 and newer) or 9091 (6.x and earlier).
        _docker_exec_with_retry "cmos" "curl -s -X POST -H \"Content-Type: application/json\" -d \
          '{\"couchbaseConfig\":{\"username\":\"$server_user\",\"password\":\"$server_pwd\"}, \
          \"hostname\":\"node$start.local\"}' 'http://localhost:7194/config/api/v1/clusters/add' || echo 'failed'" "{\"ok\":true}"

        echo ""
        echo "- Nodes added to Prometheus scrape config under the cluster"

        # Rebalance newly-added nodes into the fully provisioned cluster
        if (( to_provision > 1 )); then
            _docker_exec_with_retry "$uid" "/opt/couchbase/bin/couchbase-cli rebalance --cluster \"http://$uid.local:8091\" \
              --username \"$server_user\" --password \"$server_pwd\" --no-progress-bar --no-wait || echo 'failed'" "SUCCESS: "
        fi

        echo "- Rebalance started"

        # Load sample buckets
        _load_sample_buckets "$uid" "$load" "$server_user" "$server_pwd" "${sample_buckets[@]}"

        echo "Cluster configuration complete."
        echo "---------------------------------"
        echo ""

        local nodes_left=$((nodes_left - to_provision))

    done

    # Reload Prometheus to start scraping the added clusters
    _docker_exec_with_retry "cmos" "curl -s -X POST localhost:9090/prometheus/-/reload || echo 'failed'" ""
    echo "Refreshed Prometheus to start scraping. It may take up to a minute to fully load these stats into Grafana, and you may need to close and reopen the page."
    echo ""

}
