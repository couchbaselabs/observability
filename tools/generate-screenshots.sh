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

# Simple script take dashboard screenshots from CMOS.
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
CMOS_PORT=${CMOS_PORT:-8080}
CMOS_HOST=${CMOS_HOST:-localhost:$CMOS_PORT}
CMOS_CONTAINER_NAME=${CMOS_CONTAINER_NAME:-screenshot-cmos}

# Remove anything with the same name including all volumes
docker rm --force --volumes "$CMOS_CONTAINER_NAME" &> /dev/null

# Remove any previous screenshots
rm -fv "${SCRIPT_DIR}/../testing/screenshots/*.png"

"${SCRIPT_DIR}/build-oss-container.sh"
docker run --rm -d --name "$CMOS_CONTAINER_NAME" -p "$CMOS_PORT:8080" "$CMOS_IMAGE"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../testing/helpers/url-helpers.bash"

# Ignore the BATS fail usage that will trigger a failure anyway
wait_for_url 60 "$CMOS_HOST/grafana/api/health"

# Build and run the screenshot utility
pushd "${SCRIPT_DIR}/../testing/screenshots"
    npm install
    node index.js all
popd

# Clean up
docker stop "$CMOS_CONTAINER_NAME"
