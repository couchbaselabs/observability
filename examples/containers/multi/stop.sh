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

################
# This script stops and deletes all containers with the "cbs_server_exp" image, removes the image itself,
# and prunes any hanging images. Because these containers are started with the '--rm' flag, any associated
# anonymous volumes are also removed.
# Finally the CMOS container is removed, with its network and volumes deleted.
################
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
export CMOS_IMAGE # This is required for reference in the docker-compose file

# Delete ALL containers with the cbs_server_exp image
docker ps -a --filter "ancestor=cbs_server_exp" --format '{{.ID }}' | xargs docker rm -fv > /dev/null
echo "All cbs_server_exp containers deleted successfully."

# Delete the cbs_server_exp image - this needs to be here as we are using the CLI and so have no
# reference to containers to be able to stop them without the image name. container-clean removes the
# tag. By moving to docker-compose for nodes this will be avoided.
docker rmi -f "cbs_server_exp"
echo "cbs_server_exp image deleted."

# Remove the CMOS container
pushd "${SCRIPT_DIR}" || exit 1
    docker-compose down -v --remove-orphans
popd || exit

echo "------------------------------------"
echo "Example stopped and cleaned successfully."
