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
    echo "Verify pre-requisites"
    run : "${TEST_NAMESPACE?"Need to set TEST_NAMESPACE"}"
    assert_success

    run kubectl delete namespace "$TEST_NAMESPACE"
    kubectl create namespace "$TEST_NAMESPACE"
}

teardown() {
    if [ "$SKIP_TEARDOWN" == "true" ]; then
        skip "Skipping teardown"
    elif [ "$TEST_NATIVE" != "true" ]; then
        run helm uninstall --namespace "${TEST_NAMESPACE}" couchbase
        run kubectl delete --force --grace-period=0 --now=true -n "$TEST_NAMESPACE" -f "${BATS_TEST_DIRNAME}/../resources/default-microlith.yaml"
        run kubectl delete namespace "$TEST_NAMESPACE"
    fi
}

# These are required for bats-detik
# shellcheck disable=SC2034
DETIK_CLIENT_NAME="kubectl -n $TEST_NAMESPACE"
# shellcheck disable=SC2034
DETIK_CLIENT_NAMESPACE="${TEST_NAMESPACE}"

createDefaultDeployment() {
    # Prometheus configuration is all pulled from this directory
    kubectl create -n "$TEST_NAMESPACE" configmap prometheus-config --from-file="${BATS_TEST_DIRNAME}/../resources/prometheus/"

    # Deploy the microlith, without couchbase
    kubectl apply -n "$TEST_NAMESPACE" -f "${BATS_TEST_DIRNAME}/../resources/default-microlith.yaml"
    sleep 30
}

# Aim to use locals in case we want to parallelise to prevent overwriting globals
setupPortForwarding() {
    # Port forward into the K8S cluster
    local PORT_FORWARD_PID_FILE=$1
    kubectl -n "$TEST_NAMESPACE" port-forward svc/couchbase-grafana-http "$CMOS_PORT:8080" &
    echo "$!" > "${PORT_FORWARD_PID_FILE}"

    # Takes a little while to actually set up
    local LOCAL_SERVICE_URL="localhost:$CMOS_PORT"
    local ATTEMPTS=0
    local MAX_ATTEMPTS=6
    until curl -s -o /dev/null "$LOCAL_SERVICE_URL"; do
        # shellcheck disable=SC2086
        if [[ $ATTEMPTS -gt $MAX_ATTEMPTS ]]; then
            fail "unable to communicate with CMOS on $LOCAL_SERVICE_URL after $ATTEMPTS attempts"
        fi
        ATTEMPTS=$((ATTEMPTS+1))
        echo "Attempt $ATTEMPTS of $MAX_ATTEMPTS for CMOS on $LOCAL_SERVICE_URL"
        sleep 10
    done
}

# Test that we can do a default deployment from scratch
@test "Verify simple deployment from scratch" {
    createDefaultDeployment

    kubectl get pods --all-namespaces

    # Now check it comes up
    try "at most 10 times every 30s to find 1 pod named 'couchbase-grafana-*' with 'status' being 'running'"
    # Note this only tests that it is marked as 'running', it may then crash out so need more checks

    kubectl -n "$TEST_NAMESPACE" describe service loki
    kubectl -n "$TEST_NAMESPACE" describe service couchbase-grafana-http

    # Check we have the relevant services exposed
    verify "there is 1 service named 'couchbase-grafana-http'"
    verify "'port' is '8080' for services named 'couchbase-grafana-http'"
    verify "there is 1 service named 'loki'"
    verify "'port' is '3100' for services named 'loki'"

    # Port forward into the K8S cluster
    local PID_FILE
    PID_FILE=$(mktemp)
    setupPortForwarding "${PID_FILE}"
    local LOCAL_SERVICE_URL="localhost:$CMOS_PORT"

    # Check the web server provides the landing page
    run curl --show-error --silent "$LOCAL_SERVICE_URL"
    assert_success
    assert_output --partial 'Couchbase Observability Stack' # Check that this string is in there

    local PROMETHEUS_URL="$LOCAL_SERVICE_URL/prometheus"

    # Check we have a valid prometheus end point exposed and it is healthy
    curl --show-error --silent "$PROMETHEUS_URL/-/healthy"

    # Check we have loaded the right config: https://prometheus.io/docs/prometheus/latest/querying/api/#config
    run curl --show-error --silent "$PROMETHEUS_URL/api/v1/status/config"
    assert_success
    assert_output --partial 'couchbase-kubernetes-pods' # The default config does not contain this - we could diff as well

    # TODO:
    # Check we have no Couchbase targets but all the internal ones
    # https://prometheus.io/docs/prometheus/latest/querying/api/#targets
    run curl --show-error --silent "$PROMETHEUS_URL/api/v1/targets?state=active"
    assert_success
    # Check that alerts and rules are set up, with defaults only
    # https://prometheus.io/docs/prometheus/latest/querying/api/#rules
    run curl --show-error --silent "$PROMETHEUS_URL/api/v1/rules"
    assert_success
    # https://prometheus.io/docs/prometheus/latest/querying/api/#alerts
    run curl --show-error --silent "$PROMETHEUS_URL/api/v1/alerts"
    assert_success
    # Check that default dashboards are available
    local GRAFANA_URL="$LOCAL_SERVICE_URL/grafana"
    # https://grafana.com/docs/grafana/latest/http_api/dashboard/#gets-the-home-dashboard
    # https://grafana.com/docs/grafana/latest/http_api/dashboard/#get-dashboard-by-uid
    run curl --show-error --silent "$GRAFANA_URL/api/search"
    assert_success
    run curl --show-error --silent "$GRAFANA_URL/api/dashboards/home"
    assert_success

    # Check Loki is up
    local LOKI_URL="$LOCAL_SERVICE_URL/loki"
    run curl --show-error --silent "$LOKI_URL/ready"
    assert_success

    pkill -F "${PID_FILE}"
}

createCouchbaseCluster() {
    # Add Couchbase via helm chart
    helm repo add couchbase https://couchbase-partners.github.io/helm-charts
    helm repo update
    helm upgrade --install --debug --namespace "$TEST_NAMESPACE" --create-namespace couchbase couchbase/couchbase-operator --set cluster.image="${COUCHBASE_SERVER_IMAGE}"
    sleep 60
}

@test "Verify Couchbase Server metrics" {
    # Spin up a Couchbase cluster to then confirm we get metrics and targets for that
    createCouchbaseCluster
    createDefaultDeployment
    run kubectl get pods --all-namespaces
    try "at most 10 times every 30s to find 1 pod named 'couchbase-grafana-*' with 'status' being 'running'"
    run kubectl get pods --all-namespaces
    try "at most 10 times every 30s to find 3 pods named 'couchbase-couchbase-cluster-*' with 'status' being 'running'"
}

createLoggingCluster() {
    # Create the secret for Fluent Bit customisation
    kubectl create --namespace "$TEST_NAMESPACE" secret generic fluent-bit-custom --from-file="${TEST_KUBERNETES_RESOURCES_ROOT}/fluent-bit.conf"

    helm repo add couchbase https://couchbase-partners.github.io/helm-charts
    helm repo update
    helm upgrade --install --debug --namespace "$TEST_NAMESPACE" couchbase couchbase/couchbase-operator --values="${TEST_KUBERNETES_RESOURCES_ROOT}/helm/couchbase-cluster-logging-values.yaml"
}

@test "Verify Loki deployment from scratch" {
    skip "TODO"
    createDefaultDeployment
    # Slightly custom config to send logs
    createLoggingCluster
    # Test that we can explicitly poke the Loki API
    # Test that we can send logs to it
}

@test "Verify disabling of components in microlith" {
    run : "${TEST_CUSTOM_CONFIG?"Need to set TEST_CUSTOM_CONFIG"}"
    assert_success

    # Turn components off and confirm not available
    cat << __EOF__ | kubectl create -n "$TEST_NAMESPACE" -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: $TEST_CUSTOM_CONFIG # The default deployment uses this as an optional config map
data:
  DISABLE_LOKI: "true"
__EOF__
    # Disable Loki
    createDefaultDeployment
    sleep 10
    try "at most 10 times every 30s to find 1 pod named 'couchbase-grafana-*' with 'status' being 'running'"

    # Now check they are disabled
    # shellcheck disable=SC2046
    run kubectl logs --namespace="$TEST_NAMESPACE" $(kubectl get pods --namespace="$TEST_NAMESPACE" -o=name|grep couchbase-grafana)
    assert_success
    assert_output --partial "[ENTRYPOINT] Disabled as DISABLE_LOKI set"

    # Port forward into the K8S cluster
    local PID_FILE
    PID_FILE=$(mktemp)
    setupPortForwarding "${PID_FILE}"
    local LOCAL_SERVICE_URL="localhost:$CMOS_PORT"

    # Attempt to hit the endpoints as well
    run curl --show-error --silent "$LOCAL_SERVICE_URL"
    assert_success

    # https://grafana.com/docs/loki/latest/api/#get-ready
    run curl --show-error --silent "$LOCAL_SERVICE_URL/loki/ready"
    # assert_failure
    # Nginx reverse proxy gives us a page for a 404
    assert_output --partial "404 Not Found"

    pkill -F "${PID_FILE}"
    rm -f "${PID_FILE}"
}

@test "Verify customisation by adding" {
    skip "TODO"
    # Add new rules and check for
    # Add new dashboards and check for
}

@test "Verify default rules are triggered" {
    skip "TODO"
    # Create error conditions and ensure the rule is triggered
}