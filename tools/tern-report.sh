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

# Simple script to analyse the microlith container with Tern: https://github.com/tern-tools/tern
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

# Ensure we have the container built
"$SCRIPT_DIR/build-oss-container.sh"

if [[ -d "$SCRIPT_DIR/tern" ]]; then
    pushd "$SCRIPT_DIR/tern"
        git pull
    popd
else
    mkdir -p "$SCRIPT_DIR/tern"
    git clone https://github.com/tern-tools/tern.git "$SCRIPT_DIR/tern"
fi

# Build the Tern container
docker build -f docker/Dockerfile -t ternd "$SCRIPT_DIR/tern"
# Now run it against CMOS
docker run --privileged --device /dev/fuse -v /var/run/docker.sock:/var/run/docker.sock --rm ternd report --docker-image "$CMOS_IMAGE" > "$SCRIPT_DIR/tern-output.txt"
docker run --privileged --device /dev/fuse -v /var/run/docker.sock:/var/run/docker.sock --rm ternd report --report-format html --docker-image "$CMOS_IMAGE" > "$SCRIPT_DIR/tern-output.html"
