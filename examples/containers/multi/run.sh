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

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
export CMOS_IMAGE # This is required for reference in the docker-compose file

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

OSS_FLAG=${OSS_FLAG:-false}
oss_warning=
"The homepage of Grafana will show no data as the Cluster Manager is not part of the OSS \
container, and so the cluster overview cannot be shown. Any dashboard containing Prometheus \
metrics will still display information."

#### SCRIPT START ####

# Determine if there are any nodes with conflicting names
NODES_MATCHING=$(docker ps -a --filter "ancestor=cbs_server_exp" | grep -c '')
NODES_MATCHING=$((NODES_MATCHING-1))
if (( NODES_MATCHING > 0 )); then

  echo "------------------"
  echo "There are $NODES_MATCHING existing containers with \
image 'cbs_server_exp': ($(docker ps -a --filter "ancestor=cbs_server_exp" \
    --format '{{.Names}}' | tac | paste -s -d, -))"

  read -r -p "These nodes and the CMOS container must be destroyed to continue. Are you sure? [y/N]: " RESPONSE
    if [[ "$RESPONSE" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        "$SCRIPT_DIR"/stop.sh
        echo "Completed."
    else
        echo "Exiting. No containers were destroyed."
        exit
    fi

fi

# Build CMOS container
pushd "${SCRIPT_DIR}" || exit 1
    docker-compose up -d --force-recreate
popd || exit

# Build Couchbase Server/exporter container
docker build -f "$SCRIPT_DIR"/../../../testing/resources/containers/cb-with-exporter.Dockerfile "$SCRIPT_DIR"/helpers -t "cbs_server_exp" --build-arg VERSION="$COUCHBASE_SERVER_IMAGE"

# Create $NUM_NODES containers running Couchbase Server $VERSION and the exporter
start_new_nodes "$NUM_NODES" "cbs_server_exp"

# Initialise and partition nodes as evenly as possible into $NUM_CLUSTERS clusters, register them with CBMM
# and if $LOAD=true throw a light (non-zero) load at the cluster to simulate use using cbpillowfight
configure_servers "$NUM_NODES" "$NUM_CLUSTERS" "$SERVER_USER" "$SERVER_PWD" "$NODE_RAM" "$LOAD" "$OSS_FLAG" 

echo "All done. Go to: http://localhost:8080."

if $OSS_FLAG; then
  echo "$oss_warning"
fi