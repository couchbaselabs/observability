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

#####

# Pre-conditions: 
#   - No containers named "cmos" or "node$i" where $i is an integer up to the number
#     of nodes desired in the cluster ($NUM_NODES). Checked by the script and handled by 
#     the user, either via destroying them manually (e.g., make clean) or agreeing to them
#     being removed by the script.
#   - The couchbase/observability-stack Docker image built (handled by the Makefile)

# Post-conditions: 
#   - A single container named "cmos" with the CMOS Microlith running. 
#   - A total of $NUM_NODES containers with the specified Couchbase Server version
#     and Prometheus exporter installed, partitioned as evenly as possible into $NUM_CLUSTERS
#   - cbmultimanager configured to monitor all clusters
#   - Grafana configured to retrieve statistics from CBMM API for dashboards
set -eu

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CBS_EXP_IMAGE_NAME=${CBS_EXP_IMAGE_NAME:-"cbs_server_exp"}

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

# Disable check as checked elsewhere
# shellcheck disable=SC1091
source "$SCRIPT_DIR"/helpers/driver.sh

# Environment variables
COUCHBASE_SERVER_VERSION=${COUCHBASE_SERVER_VERSION:-6.6.3}
COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:$COUCHBASE_SERVER_VERSION}

NUM_CLUSTERS=${NUM_CLUSTERS:-3}
NUM_NODES=${NUM_NODES:-8}
WAIT_TIME=${WAIT_TIME:-60}

CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}
SERVER_USER=${SERVER_USER:-"Administrator"}
SERVER_PWD=${SERVER_PWD:-"password"}

NODE_RAM=${NODE_RAM:-1024}
LOAD=${LOAD:-true}

#### SCRIPT START ####

# Determine if there are any nodes with conflicting names
nodes_matching=$(docker ps -a --filter "ancestor=$CBS_EXP_IMAGE_NAME" | grep -c '')
nodes_matching=$((nodes_matching-1))
if (( nodes_matching > 0 )); then

  echo "------------------"
  echo "There are $nodes_matching existing containers with \
image '$CBS_EXP_IMAGE_NAME': ($(docker ps -a --filter "ancestor=$CBS_EXP_IMAGE_NAME" \
    --format '{{.Names}}' | tac | paste -s -d, -))"

  read -r -p "These nodes and the CMOS container must be destroyed to continue. Are you sure? [y/N]: " response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        "$SCRIPT_DIR"/stop.sh
        echo "Completed."
    else
        echo "Exiting. No containers were destroyed."
        exit
    fi

fi

# Build CMOS container
docker-compose -f "$SCRIPT_DIR"/docker-compose.yml up -d --force-recreate 

# Build Couchbase Server/exporter container
docker image build "$SCRIPT_DIR"/helpers -t $CBS_EXP_IMAGE_NAME --build-arg VERSION="$COUCHBASE_SERVER_IMAGE"

# Create $NUM_NODES containers running Couchbase Server $VERSION and the exporter
start_new_nodes "$NUM_NODES" "$CBS_EXP_IMAGE_NAME"

# Initialise and partition nodes as evenly as possible into $NUM_CLUSTERS clusters, register them with CBMM
# and if $LOAD=true throw a light (non-zero) load at the cluster to simulate use using cbpillowfight
configure_servers "$NUM_NODES" "$NUM_CLUSTERS" "$SERVER_USER" "$SERVER_PWD" "$NODE_RAM" "$LOAD" 

echo "All done. Go to: http://localhost:8080."

## TODO:
# Rename and put under subpath /containers 
  # Be careful as docker-compose named networks prefixed with parent folder name, this will change from native -> something else
  # The only current dependency on this name is the --network arg passed to the docker run command for the server/exporter image
# Improve enduser UX
# Rewrite: 
# - Rework /driver.sh configure_servers func to use another CBS instance to provision, decoupling
# - have an entrypoint script for each container (parallel)


# Finish UX thing "please try again?", exponential wait time for nodes failing command