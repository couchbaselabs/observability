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
DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
IMAGE=${IMAGE:-$DOCKER_USER/observability-stack-test:$DOCKER_TAG}

SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-no}

docker build -f "${SCRIPT_DIR}/../microlith-test/Dockerfile" -t "${IMAGE}" "${SCRIPT_DIR}/../microlith-test/"

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
    "${SCRIPT_DIR}/../../examples/native/run.sh"
fi

docker run --rm -it -e TEST_NATIVE=true "${IMAGE}"
