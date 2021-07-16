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
TIMEOUT=${TIMEOUT:-30}
COMPLETIONS=${COMPLETIONS:-1}
PARALLELISM=${PARALLELISM:-1}

CLUSTER_NAME=${CLUSTER_NAME:-microlith-test}
SKIP_CLUSTER_CREATION=${SKIP_CLUSTER_CREATION:-no}

docker build -f "${SCRIPT_DIR}/../microlith-test/Dockerfile" -t "${IMAGE}" "${SCRIPT_DIR}/../microlith-test/"

if [[ "${SKIP_CLUSTER_CREATION}" != "yes" ]]; then
    CLUSTER_NAME="${CLUSTER_NAME}" "${SCRIPT_DIR}/../../examples/kubernetes/run.sh"
fi

sed -e "s|%%IMAGE%%|$IMAGE|" \
    -e "s/%%TIMEOUT%%/$TIMEOUT/" \
    -e "s/%%COMPLETIONS%%/$COMPLETIONS/" \
    -e "s/%%PARALLELISM%%/$PARALLELISM/" \
    "${SCRIPT_DIR}/testing.yaml" > "${SCRIPT_DIR}/testing-actual.yaml"

kind load docker-image "${IMAGE}" --name="${CLUSTER_NAME}"

if kubectl delete -f "${SCRIPT_DIR}/testing-actual.yaml"; then
    echo "Removed previous job"
fi
kubectl apply -f "${SCRIPT_DIR}/testing-actual.yaml"

# Wait for the job to complete and grab the logs either way
exitCode=1
if kubectl wait --for=condition=complete job/microlith-test --timeout=30s; then
    # All ok so remove the job for the next run
    kubectl delete -f "${SCRIPT_DIR}/testing-actual.yaml"
    rm -f "${SCRIPT_DIR}/testing-actual.yaml"
    exitCode=0
fi

# shellcheck disable=SC2046
kubectl logs $(kubectl get pods -l job-name=microlith-test --no-headers -o custom-columns=":metadata.name")
exit $exitCode