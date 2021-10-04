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

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
IMAGE=${IMAGE:-$DOCKER_USER/observability-stack-test:$DOCKER_TAG}

SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-yes}
COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:7.0.1}
KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}
CLUSTER_NAME=${CLUSTER_NAME:-kind-$DOCKER_TAG}

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
    # Create a 4 node KIND cluster
    echo "Recreating full cluster"
    kind delete cluster

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

    kind create cluster --config="${CLUSTER_CONFIG}" --name="${CLUSTER_NAME}"
    rm -f "${CLUSTER_CONFIG}"
fi

# Wait for cluster to come up
docker pull "${COUCHBASE_SERVER_IMAGE}"
kind load docker-image "${COUCHBASE_SERVER_IMAGE}" --name="${CLUSTER_NAME}"
kind load docker-image "${IMAGE}" --name="${CLUSTER_NAME}"
kind load docker-image "${CMOS_IMAGE}" --name="${CLUSTER_NAME}"

docker run "${KUBECONFIG}":/home/.kube/config --rm -t "${IMAGE}"
