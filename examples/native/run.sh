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
#
set -x 
#
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

source "$SCRIPT_DIR"/helpers/driver.sh

## Environment variables
DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

CLUSTER_NUMBER=${CLUSTER_NUMBER:-2}

SERVER_USER=${SERVER_USER:-"Administrator"}
SERVER_PASS=${SERVER_PASS:-"couchbase"}

CB_VERSION=${CB_VERSION:-"6.6.3"}
VAGRANTS_OS=${VAGRANTS_OS:-"centos7"}
VAGRANTS_LOCATION=${VAGRANTS_LOCATION:-$HOME}
CREATE_VAGRANTS=${CREATE_VAGRANTS:-true} # Set to false if you already have configured (correct USER/PASS) CBS vagrants running

export VAGRANT_NODES=${VAGRANT_NODES:-3}
export VAGRANT_CPUS=${VAGRANT_CPUS:-1}
export VAGRANT_RAM=${VAGRANT_RAM:-1024}

#### SCRIPT START ####
docker-compose -f "${SCRIPT_DIR}"/docker-compose.yml up -d --force-recreate

if $CREATE_VAGRANTS; then
  remove_previous_vagrants "$CB_VERSION" "$VAGRANTS_OS" "$CLEAN_BOX"
  start_vagrant_cluster "$CB_VERSION" "$VAGRANTS_OS" "$VAGRANTS_LOCATION"
  configure_servers "$CB_VERSION" "$VAGRANTS_OS" "$CLUSTER_NUMBER" "$SERVER_USER" "$SERVER_PASS"
fi

echo "All done. Go to: http://localhost:8080 and sign in with admin:password."
# Open web browser automatically, go to Grafana?
# Check rescrape of mounted dashboards works and can be refreshed without rebuilding docker container