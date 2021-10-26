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

load "$HELPERS_ROOT/test-helpers.bash"

ensure_variables_set TEST_NAMESPACE CMOS_PORT TEST_CUSTOM_CONFIG

load "$BATS_DETIK_ROOT/utils.bash"
load "$BATS_DETIK_ROOT/linter.bash"
load "$BATS_DETIK_ROOT/detik.bash"
load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

setup_file() {
    # Parallel execution of port forwarding *may* cause problems so force serial (current default anyway)
    export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

setup() {
    if [ "${TEST_NATIVE:-false}" == "true" ]; then
        skip "Skipping kubernetes specific tests"
    fi

    run kubectl delete namespace "$TEST_NAMESPACE"
    kubectl create namespace "$TEST_NAMESPACE"
}

teardown() {
    if [ "${SKIP_TEARDOWN:-false}" == "true" ]; then
        skip "Skipping teardown"
    elif [ "${TEST_NATIVE:-false}" != "true" ]; then
        run pkill kubectl # Ensure we remove all port forwarding
        run helm uninstall --namespace "${TEST_NAMESPACE}" couchbase
        run kubectl delete --force --grace-period=0 --now=true -n "$TEST_NAMESPACE" -f "${BATS_TEST_DIRNAME}/resources/default-microlith.yaml"
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
    kubectl create -n "$TEST_NAMESPACE" configmap prometheus-config --from-file="${BATS_TEST_DIRNAME}/resources/prometheus/"

    # Deploy the microlith, without couchbase
    kubectl apply -n "$TEST_NAMESPACE" -f "${BATS_TEST_DIRNAME}/resources/default-microlith.yaml"
    sleep 30
}

# Aim to use locals in case we want to parallelise to prevent overwriting globals
setupPortForwarding() {
    # Port forward into the K8S cluster
    local pid_file=$1
    local local_port=$2
    kubectl -n "$TEST_NAMESPACE" port-forward svc/couchbase-grafana-http "$local_port:$CMOS_PORT" &
    echo "$!" > "${pid_file}"

    # Takes a little while to actually set up
    local local_service_url="localhost:$local_port"
    local attempts=0
    local max_attempts=6
    until curl -s -o /dev/null "$local_service_url"; do
        # shellcheck disable=SC2086
        if [[ $attempts -gt $max_attempts ]]; then
            run pkill -F "${pid_file}"
            fail "unable to communicate with CMOS on $local_service_url after $attempts attempts"
        fi
        attempts=$((attempts+1))
        echo "Attempt $attempts of $max_attempts for CMOS on $local_service_url"
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
    local pid_file
    pid_file=$(mktemp)
    local local_port=$(find_unused_port)
    setupPortForwarding "${pid_file}" "${local_port}"
    local local_service_url="localhost:${local_port}"

    # Check the web server provides the landing page
    run curl --show-error --silent "$local_service_url"
    assert_success
    assert_output --partial 'Couchbase Monitoring & Observability Stack' # Check that this string is in there

    local prometheus_url="$local_service_url/prometheus"

    # Check we have a valid prometheus end point exposed and it is healthy
    curl --show-error --silent "$prometheus_url/-/healthy"

    # Check we have loaded the right config: https://prometheus.io/docs/prometheus/latest/querying/api/#config
    run curl --show-error --silent "$prometheus_url/api/v1/status/config"
    assert_success
    assert_output --partial 'couchbase-kubernetes-pods' # The default config does not contain this - we could diff as well

    # TODO:
    # Check we have no Couchbase targets but all the internal ones
    # https://prometheus.io/docs/prometheus/latest/querying/api/#targets
    run curl --show-error --silent "$prometheus_url/api/v1/targets?state=active"
    assert_success
    # Check that alerts and rules are set up, with defaults only
    # https://prometheus.io/docs/prometheus/latest/querying/api/#rules
    run curl --show-error --silent "$prometheus_url/api/v1/rules"
    assert_success
    # https://prometheus.io/docs/prometheus/latest/querying/api/#alerts
    run curl --show-error --silent "$prometheus_url/api/v1/alerts"
    assert_success
    # Check that default dashboards are available
    local grafana_url="$local_service_url/grafana"
    # https://grafana.com/docs/grafana/latest/http_api/dashboard/#gets-the-home-dashboard
    # https://grafana.com/docs/grafana/latest/http_api/dashboard/#get-dashboard-by-uid
    run curl --show-error --silent "$grafana_url/api/search"
    assert_success
    run curl --show-error --silent "$grafana_url/api/dashboards/home"
    assert_success

    # Check Loki is up
    local loki_url="$local_service_url/loki"
    run curl --show-error --silent "$loki_url/ready"
    assert_success

    pkill -F "${pid_file}"
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
    kubectl create --namespace "$TEST_NAMESPACE" secret generic fluent-bit-custom --from-file="${BATS_TEST_DIRNAME}/resources/fluent-bit.conf"

    helm repo add couchbase https://couchbase-partners.github.io/helm-charts
    helm repo update
    helm upgrade --install --debug --namespace "$TEST_NAMESPACE" couchbase couchbase/couchbase-operator --values="${BATS_TEST_DIRNAME}/resources/helm/couchbase-cluster-logging-values.yaml"
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
    local pid_file
    pid_file=$(mktemp)
    local local_port=$(find_unused_port)
    setupPortForwarding "${pid_file}" "${local_port}"
    local local_service_url="localhost:${local_port}"

    # Attempt to hit the endpoints as well
    run curl --show-error --silent "$local_service_url"
    assert_success

    # https://grafana.com/docs/loki/latest/api/#get-ready
    run curl --show-error --silent "$local_service_url/loki/ready"
    # assert_failure
    # Nginx reverse proxy gives us a page for a 404
    assert_output --partial "404 Not Found"

    pkill -F "${pid_file}"
    rm -f "${pid_file}"
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
