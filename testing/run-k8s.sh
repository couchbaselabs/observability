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

# Run all the K8S cluster tests against a KIND cluster.
# It relies on BATS being installed, see tools/install-bats.sh
set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ "${SKIP_BATS:-no}" != "yes" ]]; then
    # No point shell checking it as done separately anyway
    # shellcheck disable=SC1091
    /bin/bash "${SCRIPT_DIR}/../tools/install-bats.sh"
fi

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/test-common.sh"
# Anything that is not common now specified:
export TEST_PLATFORM=kubernetes
export TEST_NAMESPACE=${TEST_NAMESPACE:-test}
export TEST_CUSTOM_CONFIG=${TEST_CUSTOM_CONFIG:-test-custom-config}

SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-yes}
CLUSTER_NAME=${CLUSTER_NAME:-kind-$DOCKER_TAG}

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
    # Create a 4 node KIND cluster
    echo "Recreating full cluster"
    kind delete cluster --name="${CLUSTER_NAME}"
    kind create cluster --name="${CLUSTER_NAME}" --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
EOF
fi

kubectl cluster-info

# Wait for cluster to come up
docker pull "${COUCHBASE_SERVER_IMAGE}"
kind load docker-image "${COUCHBASE_SERVER_IMAGE}" --name="${CLUSTER_NAME}"
kind load docker-image "${CMOS_IMAGE}" --name="${CLUSTER_NAME}"

# Run envsubst on all test files that might need it
while IFS= read -r -d '' INPUT_FILE; do
    OUTPUT_FILE=${INPUT_FILE%%-template.yaml}.yaml
    echo "Substitute template ${INPUT_FILE} --> ${OUTPUT_FILE}"
    # Make sure to leave alone anything that is not a defined environment variable
    # TODO: filter by TEST_ prefix
    envsubst "$(env | cut -d= -f1 | sed -e 's/^/$/')"  < "${INPUT_FILE}" > "${OUTPUT_FILE}"
done < <(find "${TEST_ROOT}/" -type f -name '*-template.yaml' -print0)

# This function will call `exit`, so any cleanup must be done inside of it.
run_tests "${1-}"
