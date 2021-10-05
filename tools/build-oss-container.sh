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

# Simple script to remove proprietary components from the microlith Dockerfile and build it.
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

# Remove everything between `# Couchbase proprietary start` and `# Couchbase proprietary end`
sed '/^# Couchbase proprietary start/,/^# Couchbase proprietary end/d' "$SCRIPT_DIR/../microlith/Dockerfile" > "$SCRIPT_DIR/../microlith/Dockerfile.oss"
docker build -t "$CMOS_IMAGE" -f "$SCRIPT_DIR/../microlith/Dockerfile.oss" "$SCRIPT_DIR/../microlith"
