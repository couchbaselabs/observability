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

# Returns the exposed port of a Docker container (usually used with a Docker Compose service).
# Arguments:
# $1: the name of the container, or part of the name
# $2: the container port to find the host counterpart of
function get_service_port() {
    ports=$(docker ps --filter "name=$1" --format "{{.Ports}}")
    echo "${ports}" | sed -e 's/, /\n/g' | perl -ne 'print "$1" if /0.0.0.0:(\d+)->'"$2"'/'
}

# Initializes a Couchbase Server cluster.
# Uses the list of nodes in $COUCHBASE_SERVER_HOSTS. Passes through the arguments in $1 to `docker run`.
# Exits immediately if it encounters an error.
# Arguments:
# $1: the path to couchbase-cli, or a command that runs it (e.g. `docker run couchbase couchbase-cli`)
function initialize_couchbase_cluster() {
  set -euo pipefail
  # cluster-init the first node
  first_node=$(echo "$COUCHBASE_SERVER_HOSTS" | head -n1)
  # shellcheck disable=SC2086
  $1 cluster-init \
    --cluster "$first_node" --cluster-username Administrator \
    --cluster-password password --cluster-ramsize 256 --cluster-fts-ramsize 256 --cluster-index-ramsize 256 \
    --cluster-eventing-ramsize 256 --cluster-analytics-ramsize 1024 --cluster-name "CMOS Smoke Cluster" \
    --services data,index,query --update-notifications 0

  # Add the remaining nodes to it
  # shellcheck disable=SC2086
  $1 server-add \
        --cluster "$first_node" --username Administrator --password password \
        --server-add "$(echo "$COUCHBASE_SERVER_HOSTS" | tail -n +2 | sed -e 's/:8091/:18091/g' | paste -sd ',' -)" \
        --server-add-username Administrator --server-add-password password \
        --services data,index,query
}
