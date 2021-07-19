#!/usr/bin/env bats

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

# The intention of this file is to verify the tooling installed within the container.
# This is so that it can then be used by actual tests.

load "$BATS_DETIK_ROOT/utils.bash"
load "$BATS_DETIK_ROOT/linter.bash"
load "$BATS_DETIK_ROOT/detik.bash"
load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

setup() {
    if [ "$TEST_NATIVE" == "true" ]; then
        skip "Skipping kubernetes specific tests"
    fi

    kubectl delete namespace $TEST_NAMESPACE || true
}

teardown() {
    kubectl delete namespace $TEST_NAMESPACE || true
}

TEST_NAMESPACE=${TEST_NAMESPACE:-test}
DETIK_CLIENT_NAMESPACE=${TEST_NAMESPACE}
TEST_KUBERNETES_RESOURCES_ROOT=${TEST_KUBERNETES_RESOURCES_ROOT:-/home/testing/kubernetes/resources}

# Test that we can do a default deployment from scratch
@test "Verify simple deployment from scratch" {
    DETIK_CLIENT_NAME="kubectl -n $TEST_NAMESPACE"
    kubectl create namespace "$TEST_NAMESPACE"

    # Prometheus configuration is all pulled from this directory
    kubectl create -n "$TEST_NAMESPACE" configmap prometheus-config --from-file="$TEST_KUBERNETES_RESOURCES_ROOT/prometheus/"

    # Deploy the microlith, without couchbase
    kubectl apply -n "$TEST_NAMESPACE" -f "$TEST_KUBERNETES_RESOURCES_ROOT/default-microlith.yaml"
    sleep 10

    # Now check it comes up
    try "at most 5 times every 30s to find 1 pod named 'couchbase-grafana-*' with 'status' being 'running'"
    # Note this only tests that it is marked as 'running', it may then crash out so need more checks

    # Make sure everything starts up for logs
    sleep 20

    # Export logs - can trigger a hang: https://bats-core.readthedocs.io/en/latest/writing-tests.html#file-descriptor-3-read-this-if-bats-hangs
    # kubectl logs --namespace="$TEST_NAMESPACE" $(kubectl get pods --namespace="$TEST_NAMESPACE" -o=name) >&3

    # Check we have the relevant services exposed
    verify "there is 1 service named 'couchbase-grafana-http'"
    verify "'port' is '8080' for services named 'couchbase-grafana-http'"
    verify "there is 1 service named 'loki'"
    verify "'port' is '3100' for services named 'loki'"

    # Check we have a valid prometheus end point exposed
}

@test "Verify Loki deployment from scratch" {
    # Slightly custom config to send logs
    # Test that we can explicitly poke the Loki API
    # Test that we can send logs to it
}

