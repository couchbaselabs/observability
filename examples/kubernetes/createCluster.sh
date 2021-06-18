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
set -eux

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

CONFIG_DIR=$(mktemp -d)
CLUSTER_NAME=${CLUSTER_NAME:-microlith-test}
CLUSTER_CONFIG="${CONFIG_DIR}/multinode-cluster-conf.yaml"

SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-no}
REBUILD_ALL=${REBUILD_ALL:-yes}

SERVER_IMAGE=${SERVER_IMAGE:-couchbase/server:6.6.2}
SERVER_COUNT=${SERVER_COUNT:-3}

DOCKER_TAG=${DOCKER_TAG:-v1}
OPERATOR_VERSION=${OPERATOR_VERSION:-$DOCKER_TAG}
DAC_VERSION=${DAC_VERSION:-$DOCKER_TAG}

OPERATOR_REPO_DIR=${OPERATOR_REPO_DIR:-${SCRIPT_DIR}/couchbase-operator}

set +x

if [[ "${REBUILD_ALL}" == "yes" ]]; then
  echo "Full rebuild"
  SKIP_CLUSTER_CREATION=no

  if [[ ! -d "${OPERATOR_REPO_DIR}" ]]; then
    git clone --depth 1 git@github.com:couchbase/couchbase-operator.git "${OPERATOR_REPO_DIR}"
  fi

  pushd "${OPERATOR_REPO_DIR}"
  make && make container
  popd
fi

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
  echo "Recreating full cluster"

  # Simple script to deal with running up a test cluster for KIND for developing logging updates for.
  cat << EOF > "${CLUSTER_CONFIG}"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
 EphemeralContainers: true
nodes:
- role: control-plane
EOF

    for i in $(seq "$SERVER_COUNT"); do
      echo "Adding worker $i"
      cat << EOF >> "${CLUSTER_CONFIG}"
- role: worker
EOF
    done

    kind delete cluster --name="${CLUSTER_NAME}"
    kind create cluster --name="${CLUSTER_NAME}" --config="${CLUSTER_CONFIG}"
    echo "$(date) waiting for cluster..."
    until kubectl cluster-info;  do
        echo -n "."
        sleep 2
    done
    echo -n " done"

    # Check we can use the storage ok
    if ! kubectl get sc standard -o yaml|grep -q "volumeBindingMode: WaitForFirstConsumer"; then
        echo "Standard storage class is not lazy binding so needs manual set up"
        exit 1
    fi

    # Ensure we have everything we need
    kind load docker-image "couchbase/couchbase-operator:${OPERATOR_VERSION}" --name="${CLUSTER_NAME}"
    kind load docker-image "couchbase/couchbase-operator-admission:${DAC_VERSION}" --name="${CLUSTER_NAME}"
    kind load docker-image "couchbase-observability:latest" --name="${CLUSTER_NAME}"

    # Not strictly required but improves caching performance
    docker pull "${SERVER_IMAGE}"
    kind load docker-image "${SERVER_IMAGE}" --name="${CLUSTER_NAME}"
    # It also slows down everything to allow the cluster to come up fully

    rm -rf "${CONFIG_DIR}"

    # Install CRD, DAC and operator
    kubectl apply -f "${OPERATOR_REPO_DIR}/example/crd.yaml"
    "${OPERATOR_REPO_DIR}/build/bin/cbopcfg" create admission --image="couchbase/couchbase-operator-admission:${OPERATOR_VERSION}" --log-level=debug
    "${OPERATOR_REPO_DIR}/build/bin/cbopcfg" create operator --image="couchbase/couchbase-operator:${DAC_VERSION}" --log-level=debug

    # Need to wait for operator and DAC to start up
    echo "Waiting for DAC to complete..."
    until kubectl rollout status deployment couchbase-operator-admission; do
        echo -n "."
        sleep 2
    done
    echo " done"
    echo "Waiting for operator to complete..."
    until kubectl rollout status deployment couchbase-operator; do
        echo -n "."
        sleep 2
    done
    echo " done"
fi #SKIP_CLUSTER_CREATION

kubectl apply -f "${SCRIPT_DIR}/couchbase.yaml"
# kubectl apply -f "${SCRIPT_DIR}/microlith.yaml"
kubectl run couchbase-grafana --image=couchbase-observability:latest

# Wait for deployment to complete
echo "Waiting for CB to start up..."
until [[ $(kubectl get pods --field-selector=status.phase=Running --selector='app=couchbase' --no-headers 2>/dev/null |wc -l) -eq $SERVER_COUNT ]]; do
    echo -n '.'
    sleep 2
done
echo "CB started"

# TODO: run local container as well that talks to the cluster