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

source "$SCRIPT_DIR"/helpers/vagrants_up.sh

declare -A BOX_NAMES
BOX_NAMES['ubuntu10']='ubuntu-server-10044-x64-vbox4210'
BOX_NAMES['ubuntu14']='ubuntu/trusty64'
BOX_NAMES['ubuntu16']='puppetlabs/ubuntu-16.04-64-puppet'
BOX_NAMES['ubuntu18']='generic/ubuntu1804'
BOX_NAMES['ubuntu20']='generic/ubuntu2004'
BOX_NAMES['debian7']='cargomedia/debian-7-amd64-default'
BOX_NAMES['box_name']='centos5u8_x64'
BOX_NAMES['centos6']='puppetlabs/centos-6.6-64-puppet'
BOX_NAMES['centos6u4']='hansode/centos-6.4-x86_64'
BOX_NAMES['centos7']='puppetlabs/centos-7.0-64-puppet'
BOX_NAMES['centos8']='saphyre/centos-8-puppet-x86_64'
BOX_NAMES['windows']='emyl/win2008r2'
BOX_NAMES['opensuse11']='minesense/opensuse11.1'
BOX_NAMES['opensuse12']='opensuse-12.3-64'
BOX_NAMES['debian8']='lazyfrosch/debian-8-jessie-amd64-puppet'
BOX_NAMES['debian9']='generic/debian9'
BOX_NAMES['debian10']='debian/contrib-buster64'

# Remove all vagrants: vagrant global-status | awk '$1 ~ /[0-9,a-f]{6}/{system("vagrant destroy -f "$1)}'
# Remove all boxes: vagrant box list | cut -f 1 -d ' ' | xargs -L 1 vagrant box remove -f --all

# Parameters:
# $1: The Couchbase Server version to remove
# $2: The OS the version to remove runs on ($1 and $2 are combined to uniquely identify a Vagrants cluster)
# $3: Boolean specifying whether the box should be destroyed and redownloaded (in case of config persistence issues)
function remove_previous_vagrants() {

    local CB_VERSION=$1
    local VAGRANT_OS=$2
    local CLEAN_BOX=$3

    echo "Destroying all vagrants associated with Couchbase Version $CB_VERSION and Vagrants Operating System $VAGRANT_OS..."
    vagrant global-status --prune | grep "$CB_VERSION/$VAGRANT_OS" | awk '$1 ~ /[0-9,a-f]{6}/{system("vagrant destroy -f "$1)}'
    # --prune updates cached list first

    if $CLEAN_BOX; then
        echo "[CLEANING] Removing the box for Operating System $VAGRANTS_OS"
        vagrant box remove ${BOX_NAMES[$VAGRANTS_OS]} -f --all
    fi

}

# Parameters:
# $1: The Couchbase Server version to test
# $2: The OS the Vagrants cluster uses
# $3: The number of clusters to split nodes into
# $4: The Couchbase Server username to set
# $5: The Couchbase Server password to set
function configure_servers() {
    
    local CB_VERSION=$1
    local VAGRANT_OS=$2
    local CLUSTER_NUMBER=$3
    local SERVER_USER=$4
    local SERVER_PASS=$5

    local DATA_ALLOC 
    local INDEX_ALLOC
    DATA_ALLOC=$(awk -v n="$VAGRANT_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')
    INDEX_ALLOC=$(awk -v n="$VAGRANT_RAM" 'BEGIN {printf "%.0f\n", (n*0.7)}')

    all_nodes=($(vagrant global-status | grep "$CB_VERSION/$VAGRANT_OS" | awk '$1 ~ /[0-9,a-f]{6}/{system("echo "$1)}'))
    nodes_left=$VAGRANT_NODES

    for ((i=0; i<CLUSTER_NUMBER; i++))
    do

        to_provision=$(( nodes_left / (CLUSTER_NUMBER - i) )) # This is always integer division, Bash does not support decimals
        start=$(( VAGRANT_NODES - nodes_left ))
        first_uid=${all_nodes[$start]}
        
        for uid in "${all_nodes[@]:start:to_provision}" # Slice of length $to_provision, beginning at $start
            do
                
                if [[ $uid == "$first_uid" ]]; then
                    # Create and configure new cluster
                    ip=$(vagrant ssh "$uid" -c "hostname -I | cut -d' ' -f2" -- -q | tail -n1)
                    ip="${ip%%[[:cntrl:]]}" # Remove \r from IP string

                    vagrant ssh "$uid" -c "/opt/couchbase/bin/couchbase-cli cluster-init -c $ip  --cluster-username=$SERVER_USER \
                        --cluster-password=$SERVER_PASS --cluster-ramsize=$DATA_ALLOC --cluster-index-ramsize=$INDEX_ALLOC --services=data"

                    # Load sample buckets and register cluster with CBMM
                    curl -X POST -u "$SERVER_USER":"$SERVER_PASS" http://"$ip:8091"/sampleBuckets/install -d '["travel-sample", "beer-sample"]'
                    curl -u admin:password -X POST -d "{\"user\":\"$SERVER_USER\",\"password\":\"$SERVER_PASS\", \"host\":\"$ip:8091\"}" \
                        'http://localhost:8080/couchbase/api/v1/clusters'

                    # Start cbpillowfight to simulate a non-zero load (NOT stress test)
                    vagrant ssh "$uid" -c " /opt/couchbase/bin/cbc-pillowfight -u $SERVER_USER -P $SERVER_PASS -U couchbase://localhost/beer-sample \
                        -B 100 -I 1000 --rate-limit 100 &"

                    vagrant ssh "$uid" -c " /opt/couchbase/bin/cbc-pillowfight -u $SERVER_USER -P $SERVER_PASS -U couchbase://localhost/travel-sample \
                        -B 100 -I 1000 --rate-limit 100 &"

                else
                    # Add server to existing cluster
                    to_add_ip=$(vagrant ssh "$uid" -c "hostname -I | cut -d' ' -f2" -- -q | tail -n1)
                    to_add_ip="${to_add_ip%%[[:cntrl:]]}"

                    vagrant ssh "$uid" -c "/opt/couchbase/bin/couchbase-cli node-init --cluster $ip --username $SERVER_USER --password $SERVER_PASS \
                        && /opt/couchbase/bin/couchbase-cli server-add -c $ip --username $SERVER_USER --password $SERVER_PASS --server-add $to_add_ip \
                        --server-add-username $SERVER_USER --server-add-password $SERVER_PASS --services index,query"
                fi
            done

        # Rebalance fully provisioned cluster
        if (( to_provision > 1 )); then
            vagrant ssh "$uid" -c "/opt/couchbase/bin/couchbase-cli rebalance --cluster $to_add_ip --username $SERVER_USER --password $SERVER_PASS \
            --no-progress-bar --no-wait"
        fi

        nodes_left=$((nodes_left - to_provision))
    done

}
