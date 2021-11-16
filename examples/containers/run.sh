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
set -eu

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:7.0.2}

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

# Ensure we build the container locally first otherwise make
# sure one is tagged as above CMOS_IMAGE for use in the .env file.
if [[ "${SKIP_CONTAINER_BUILD:-yes}" != "yes" ]]; then
    echo "Building CMOS container"
    make -C "${SCRIPT_DIR}/../.." container
fi

rm -rf "${SCRIPT_DIR}"/logs/*.log
pushd "${SCRIPT_DIR}" || exit 1
    docker-compose up -d --force-recreate
popd || exit
