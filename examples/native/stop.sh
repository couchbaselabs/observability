#!/bin/bash
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
set -u
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Delete ALL containers with the cbs_server_exp image
# This needs to be first because docker-compose down attempts to remove the network
# and if these containers are still up and connected to it that won't be allowed (?)
docker ps -a --filter 'ancestor=cbs_server_exp' --format '{{.ID }}' | xargs docker rm -f

# Remove the CMOS container
docker-compose -f "$SCRIPT_DIR"/docker-compose.yml down -v --remove-orphans

# Transient "Error response from daemon: error while removing network: ... has active endpoints"
# - only fix is to restart Docker daemon (hanging endpoint but zero exist in inspect output so 
# cannot be manually removed)

