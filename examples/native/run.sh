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

set -eu
set -x
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

# shellcheck disable=SC1091
source "$SCRIPT_DIR"/helpers/driver.sh

## Environment variables
CLUSTER_NUM=${CLUSTER_NUM:-3}
NODE_NUM=${NODE_NUM:-8}
WAIT_TIME=${WAIT_TIME:-60}

SERVER_USER=${SERVER_USER:-"Administrator"}
SERVER_PASS=${SERVER_PASS:-"password"}

CB_VERSION=${CB_VERSION:-"enterprise-6.6.3"}
NODE_RAM=${NODE_RAM:-1024}

WAIT_TIME=${WAIT_TIME:-10}

#### SCRIPT START ####
docker-compose -f "$SCRIPT_DIR"/docker-compose.yml up -d --force-recreate
docker image build "$SCRIPT_DIR"/helpers -t "cbs_server_exp" --build-arg VERSION="$CB_VERSION"

# Remove previous nodes matching image name "cbs_server_exp"
docker ps -a | awk '{ print $1,$2 }' | grep "cbs_server_exp" | awk '{print $1 }' | xargs -I {} docker rm {} -f
start_new_nodes "$NODE_NUM"

sleep "$WAIT_TIME"
configure_servers "$NODE_NUM" "$CLUSTER_NUM" "$SERVER_USER" "$SERVER_PASS" "$NODE_RAM" 

echo "All done. Go to: http://localhost:8080 -> Grafana"