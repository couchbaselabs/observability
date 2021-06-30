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

CLUSTER_NAME=${CLUSTER_NAME:-microlith-test}
SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-no}
SERVER_IMAGE=${SERVER_IMAGE:-couchbase/server:6.6.2}

set +x

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
  echo "Recreating full cluster"
  kind delete cluster --name="${CLUSTER_NAME}"

  # Simple script to deal with running up a test cluster for KIND for developing logging updates for.
  CLUSTER_CONFIG=$(mktemp)
  cat << EOF > "${CLUSTER_CONFIG}"
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
- role: worker
- role: worker
EOF

  kind create cluster --name="${CLUSTER_NAME}" --config="${CLUSTER_CONFIG}"
  rm -f "${CLUSTER_CONFIG}"

  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/kind/deploy.yaml
  kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=120s

  # The webhook installation is not complete so just remove
  kubectl delete validatingwebhookconfigurations ingress-nginx-admission
fi #SKIP_CLUSTER_CREATION

# Build and deploy the microlith
DOCKER_BUILDKIT=1 docker build --ssh default -t couchbase-observability:v1 -f "${SCRIPT_DIR}/../../microlith/Dockerfile" "${SCRIPT_DIR}/../../microlith/"
kind load docker-image couchbase-observability:v1 --name="${CLUSTER_NAME}"
kubectl apply -f "${SCRIPT_DIR}/microlith.yaml"

# Create the secret for Fluent Bit customisation
kubectl delete secret fluent-bit-custom 2>/dev/null || true
kubectl create secret generic fluent-bit-custom --from-file="${SCRIPT_DIR}/fluent-bit.conf"

# Output the contents of the secret so we can verify
# shellcheck disable=SC2016
kubectl get secret fluent-bit-custom -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'

# Add Couchbase via helm chart
if ! helm repo add couchbase https://couchbase-partners.github.io/helm-charts; then
  if ! helm repo list|grep couchbase|grep -q https://couchbase-partners.github.io/helm-charts ; then
    echo "Unable to add Couchbase helm repository, remove the current 'couchbase' entry using 'helm repo remove couchbase' then re-run this script"
    helm repo list|grep couchbase
    exit 1
  fi
fi
helm repo update
helm upgrade --install couchbase couchbase/couchbase-operator --set cluster.image="${SERVER_IMAGE}" --values="${SCRIPT_DIR}/custom-values.yaml"

# Wait for deployment to complete
echo "Waiting for CB to start up..."
until [[ $(kubectl get pods --field-selector=status.phase=Running --selector='app=couchbase' --no-headers 2>/dev/null |wc -l) -eq 3 ]]; do
    echo -n '.'
    sleep 2
done
echo "CB running"

echo "To monitor go to http://localhost/"