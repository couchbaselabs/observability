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
COS_IMAGE=${IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
IMAGE=${IMAGE:-$DOCKER_USER/observability-stack-test:$DOCKER_TAG}
TIMEOUT=${TIMEOUT:-30}
COMPLETIONS=${COMPLETIONS:-1}
PARALLELISM=${PARALLELISM:-1}

CLUSTER_NAME=${CLUSTER_NAME:-microlith-test}
SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-yes}
COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:6.6.2}

docker build -f "${SCRIPT_DIR}/../microlith-test/Dockerfile" -t "${IMAGE}" "${SCRIPT_DIR}/../microlith-test/"

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
    # Create a 4 node KIND cluster
    echo "Recreating full cluster"
    kind delete cluster --name="${CLUSTER_NAME}"

    CLUSTER_CONFIG=$(mktemp)
    cat << EOF > "${CLUSTER_CONFIG}"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
EOF

    kind create cluster --name="${CLUSTER_NAME}" --config="${CLUSTER_CONFIG}"
    rm -f "${CLUSTER_CONFIG}"
fi

    # Wait for cluster to come up
    docker pull "${COUCHBASE_SERVER_IMAGE}"
    kind load docker-image "${COUCHBASE_SERVER_IMAGE}" --name="${CLUSTER_NAME}"

sed -e "s|%%IMAGE%%|$IMAGE|" \
    -e "s/%%TIMEOUT%%/$TIMEOUT/" \
    -e "s/%%COMPLETIONS%%/$COMPLETIONS/" \
    -e "s/%%PARALLELISM%%/$PARALLELISM/" \
    -e "s|%%COUCHBASE_SERVER_IMAGE%%|$COUCHBASE_SERVER_IMAGE|" \
    -e "s|%%COS_IMAGE%%|$COS_IMAGE|" \
    "${SCRIPT_DIR}/testing.yaml" > "${SCRIPT_DIR}/testing-actual.yaml"

kind load docker-image "${IMAGE}" --name="${CLUSTER_NAME}"
kind load docker-image "${COS_IMAGE}" --name="${CLUSTER_NAME}"

if kubectl delete -f "${SCRIPT_DIR}/testing-actual.yaml"; then
    echo "Removed previous job"
fi
kubectl apply -f "${SCRIPT_DIR}/testing-actual.yaml"

# Wait for the job to complete and grab the logs either way
exitCode=1
if kubectl wait --for=condition=ready pod/microlith-test --timeout=30s; then
    exitCode=0
fi

kubectl logs microlith-test -f
exit $exitCode