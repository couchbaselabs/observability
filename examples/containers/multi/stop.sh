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
docker ps -a --filter "ancestor=cbs_server_exp" --format '{{.ID }}' | xargs docker rm -f
echo "All cbs_server_exp containers deleted successfully."

# Delete the cbs_server_exp image - this needs to be here as we are using the CLI and so have no
# reference to containers to be able to stop them without the image name. container-clean removes the 
# tag. By moving to docker-compose for nodes this will be avoided.
docker rmi "cbs_server_exp" -f && docker image prune --force

# Remove the CMOS container
docker-compose -f "$SCRIPT_DIR"/docker-compose.yml down -v --remove-orphans

# Tidy up dev variables if they exist
DIR="$SCRIPT_DIR/../../../microlith/grafana/provisioning/dashboards"
mv -f "$DIR"/dashboard.yml.bak "$DIR"/dashboard.yml > /dev/null 2>&1

