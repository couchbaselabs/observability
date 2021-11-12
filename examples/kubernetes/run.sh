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

CLUSTER_NAME=${CLUSTER_NAME:-kind}
COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:7.0.2}

DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}
CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}

if [[ "${SKIP_CLUSTER_CREATION:-no}" != "yes" ]]; then
  echo "Recreating full cluster"

  kind delete cluster --name="${CLUSTER_NAME}"

  # Simple script to deal with running up a test cluster for KIND.
  # We use a single worker and control node here to show how to add more.
  # Locally resources are shared anyway between all nodes so unless you
  # have anti-affinity rules or similar reasons for wanting multiple nodes
  # then there is not much point.
  # We also need to set up some port mappings for ingress.
  kind create cluster --name="${CLUSTER_NAME}" --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
EOF

  # Wait for cluster to come up
  docker pull "${COUCHBASE_SERVER_IMAGE}"
  kind load docker-image "${COUCHBASE_SERVER_IMAGE}" --name="${CLUSTER_NAME}"
fi #SKIP_CLUSTER_CREATION

# Deploy kube-state-metrics via helm chart
# TODO: this should all be part of CMOS to auto-deploy via a checkbox as we have Helm in the container so it is just RBAC to sort.
# https://issues.couchbase.com/browse/CMOS-47
# https://issues.couchbase.com/browse/CMOS-80
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm upgrade --install kube-state-metrics prometheus-community/kube-state-metrics

# Prometheus configuration is all pulled from this directory
kubectl delete configmap prometheus-config || true
kubectl create configmap prometheus-config --from-file="${SCRIPT_DIR}/prometheus/custom/"

# Deploy the microlith
kind load docker-image "${CMOS_IMAGE}" --name="${CLUSTER_NAME}"
kubectl apply -f "${SCRIPT_DIR}/microlith.yaml"

# Set up ingress
INGRESS_VERSION=$(curl https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/stable.txt)
kubectl apply -f "https://raw.githubusercontent.com/kubernetes/ingress-nginx/${INGRESS_VERSION}/deploy/static/provider/kind/deploy.yaml"
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
kubectl apply -f "${SCRIPT_DIR}/ingress.yaml"

# Add Couchbase via helm chart
if ! helm repo add couchbase https://couchbase-partners.github.io/helm-charts; then
  if ! helm repo list|grep couchbase|grep -q https://couchbase-partners.github.io/helm-charts ; then
    echo "Unable to add Couchbase helm repository, remove the current 'couchbase' entry using 'helm repo remove couchbase' then re-run this script"
    helm repo list|grep couchbase
    exit 1
  fi
fi
helm repo update
helm upgrade --install couchbase couchbase/couchbase-operator --set cluster.image="${COUCHBASE_SERVER_IMAGE}" --values="${SCRIPT_DIR}/custom-values.yaml"

# Wait for deployment to complete, the Helm defaults are for a 3 pod cluster in the default namespace.
echo "Waiting for CB to start up..."
until [[ $(kubectl get pods --field-selector=status.phase=Running --selector='app=couchbase' --no-headers 2>/dev/null |wc -l) -eq 3 ]]; do
    echo -n '.'
    sleep 2
done
echo "CB running"
echo "To monitor go to http://localhost/"
