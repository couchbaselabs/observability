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
docker container rm --force --volumes "$CMOS_CONTAINER_NAME" &> /dev/null

# Remove any previous screenshots
rm -fv "${SCRIPT_DIR}"/../testing/screenshots/*.png

# Make sure to run it with sufficient configuration to include your actual data within the retention policy
docker run -d --name "$CMOS_CONTAINER_NAME" \
        -p "$CMOS_PORT:8080" \
        -v "${SCRIPT_DIR}/../testing/screenshots/data/prometheus_data_snapshot.zip:/data_snapshot/prometheus_data_snapshot.zip:ro" \
        -v "$SCRIPT_DIR/generate-screenshots-entrypoint.sh:/entrypoints/generate-screenshots-entrypoint.sh:ro" \
        -e DISABLE_PROMETHEUS="true" \
        -e PROMETHEUS_RETENTION_TIME="1y" \
        "$CMOS_IMAGE"

# Define a custom fail function for use with the BATS framework calls below - useful when it just bombs out in the container.
function fail() {
    echo "CMOS screenshot extraction FAILED"
    docker ps
    docker logs "$CMOS_CONTAINER_NAME"
    exit 1
}

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../testing/helpers/url-helpers.bash"

# Ignore the BATS fail usage that will trigger a failure anyway
wait_for_url 120 "$CMOS_HOST/grafana/api/health"

# Build and run the screenshot utility
pushd "${SCRIPT_DIR}/../testing/screenshots"
    # Any extra stuff we need to add to the Grafana URL, e.g. time period. Set empty to disable, e.g. for local testing with live data.
    export ADDITIONAL_QUERY_ARGS=${ADDITIONAL_QUERY_ARGS:-"?from=1635496105747&to=1635510441609"}
    npm install
    if [[ "${GITHUB_ACTIONS:-false}" != "true" ]]; then
        echo "Running outside of an action so generating all screenshots"
        node index.js all
    else
        echo "Running under action"
        node index.js
    fi
popd

# Clean up and ignore errors now
docker container rm --force --volumes "$CMOS_CONTAINER_NAME"
