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

function start_new_nodes() {

    local NODE_NUM=$1

    for ((i=0; i<NODE_NUM; i++))
    do
        docker run -d --name "node$i" "cbs_server_exp"
    done

}


function configure_servers() {

    local NODE_NUM=$1
    local CLUSTER_NUM=$2
    local SERVER_USER=$3
    local SERVER_PASS=$4
    local NODE_RAM=$5

    local DATA_ALLOC 
    local INDEX_ALLOC
    DATA_ALLOC=$(awk -v n="$NODE_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')
    INDEX_ALLOC=$(awk -v n="$NODE_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')

    local NAMES=(A B C D E F G H I J K L M N O P Q R S T U V W X Y Z)
    local nodes_left=$NODE_NUM

    for ((i=0; i<CLUSTER_NUM; i++))
    do

        local to_provision=$(( nodes_left / (CLUSTER_NUM - i) )) # This is always integer division, Bash does not support decimals
        local start=$(( NODE_NUM - nodes_left ))
        
        for ((j=start; j<start+to_provision; j++))
        do 

            local uid="node$j"

            if (( j == start )); then
                # Create and configure new cluster
                local ip
                ip=$(docker container inspect -f '{{ .NetworkSettings.IPAddress }}' $uid)

                local x=$(( 26 - (26 - (NODE_NUM-nodes_left)) ))
                local clust_name="Cluster ${NAMES[$x]}"

                docker exec "$uid" /opt/couchbase/bin/couchbase-cli cluster-init -c localhost --cluster-name="$clust_name" --cluster-username="$SERVER_USER" \
                    --cluster-password="$SERVER_PASS" --cluster-ramsize="$DATA_ALLOC" --cluster-index-ramsize="$INDEX_ALLOC" --services=data

                # Load sample buckets and register cluster with CBMM
                docker exec "$uid" curl -X POST -u "$SERVER_USER":"$SERVER_PASS" http://"localhost:8091"/sampleBuckets/install -d '["travel-sample", "beer-sample"]'
                docker exec cmos curl -u admin:password -X POST -d "{\"user\":\"$SERVER_USER\",\"password\":\"$SERVER_PASS\", \"host\":\"http://$ip:8091\"}" \
                        'http://localhost:8080/couchbase/api/v1/clusters'
                # Start cbpillowfight to simulate a non-zero load (NOT stress test)
                
                if $LOAD; then
                    (sleep 20 && docker exec "$uid" /opt/couchbase/bin/cbc-pillowfight -u "$SERVER_USER" -P "$SERVER_PASS" -U couchbase://localhost/beer-sample \
                        -B 100 -I 1000 --rate-limit 100 &)

                    (sleep 30 && docker exec "$uid" /opt/couchbase/bin/cbc-pillowfight -u "$SERVER_USER" -P "$SERVER_PASS" -U couchbase://localhost/travel-sample \
                        -B 100 -I 1000 --rate-limit 100 &)
                fi

            else
                # Add server to existing cluster
                local to_add_ip
                to_add_ip=$(docker container inspect -f '{{ .NetworkSettings.IPAddress }}' $uid)

                docker exec "$uid" /opt/couchbase/bin/couchbase-cli node-init --cluster "$ip" --username "$SERVER_USER" --password "$SERVER_PASS"
                docker exec "$uid" /opt/couchbase/bin/couchbase-cli server-add -c "$ip" --username "$SERVER_USER" --password "$SERVER_PASS" \
                    --server-add "$to_add_ip" --server-add-username "$SERVER_USER" --server-add-password "$SERVER_PASS" --services index,query
            fi
        done

        # Rebalance fully provisioned cluster
        if (( to_provision > 1 )); then
            docker exec "$uid" /opt/couchbase/bin/couchbase-cli rebalance --cluster "$to_add_ip" --username "$SERVER_USER" --password "$SERVER_PASS" \
            --no-progress-bar --no-wait
        fi

        local nodes_left=$((nodes_left - to_provision))
    done

}