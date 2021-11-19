#!/usr/bin/env bash

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
set -eo pipefail

# shellcheck disable=SC1091
source "$HELPERS_ROOT/url-helpers.bash"
# shellcheck disable=SC1091
source "$HELPERS_ROOT/native-helpers.bash"
# shellcheck disable=SC1091
source "$HELPERS_ROOT/couchbase-helpers.bash"

# Verifies if all the given variables are set, and exits otherwise
# Parameters:
# Variadic: variable names to check presence of
function ensure_variables_set() {
    missing=""
    for var in "$@"; do
        if [ -z "${!var}" ]; then
            missing+="$var "
        fi
    done
    if [ -n "$missing" ]; then
        if [[ $(type -t fail) == function ]]; then
            fail "Missing required variables: $missing"
        else
            echo "Missing required variables: $missing" >&2
            exit 1
        fi
    fi
}

# Finds a random, unused port on the system and echos it.
# Returns 1 and echos -1 if it can't find one.
# Have to do it this way to prevent variable shadowing.
function find_unused_port() {
    local portnum
    while true; do
        portnum=$(shuf -i 1025-65535 -n 1)
        if ! lsof -Pi ":$portnum" -sTCP:LISTEN; then
            echo "$portnum"
            return 0
        fi
    done
    echo -1
    return 1
}

# Checks if the Couchbase Server version the tests are using is less than the given version ($1).
function cb_version_lt() {
  # this will work until CBS 10, which won't be for a good few years
  [[ "$COUCHBASE_SERVER_VERSION" < "$1" ]]
  return
}

# Checks if the Couchbase Server version the tests are using greater than or equal to the given version ($1).
function cb_version_gte() {
  # this will work until CBS 10, which won't be for a good few years
  [[ ! "$COUCHBASE_SERVER_VERSION" < "$1" ]]
  return
}

# Downloads a diagnostics tarball from the test environment's CMOS.
# Can be disabled by setting DISABLE_CMOSINFO=true.
# Does nothing if CMOS_HOST is not set.
function capture_cmosinfo() {
  if [ "${DISABLE_CMOSINFO:-false}" == "true" ]; then
    echo "# Not capturing cmosinfo; DISABLE_CMOSINFO is set."
    return
  fi
  if [ -z "${CMOS_HOST:-}" ]; then
    echo "# Not capturing cmosinfo; CMOS_HOST not set."
    return
  fi
  local output
  output=$(curl -X POST -sS "http://$CMOS_HOST/config/api/v1/collectInformation")

  local filename
  filename=$(echo "$output" | sed -En -e 's/Collected support information at \/tmp\/support\/(.+)\.$/\1/p')

  local output_path="$DIAGNOSTICS_ROOT/cmosinfo"
  mkdir -p "$output_path"
  wget -q -O "$output_path/$filename" "http://$CMOS_HOST/support/$filename"
  echo "# Collected cmosinfo; saved to $output_path/$filename."
}

# Starts a docker container named `cmos` and configures $COUCHBASE_SERVER_HOSTS with Prometheus.
# Exposes a variable $CMOS_HOST with the nginx host:port.
# All parameters will be passed on to docker before the image.
function _start_cmos() {
    docker run --rm -d -p '8080' --name cmos "$@" "$CMOS_IMAGE"

    local cmos_port
    cmos_port=$(docker inspect cmos -f '{{with index .NetworkSettings.Ports "8080/tcp"}}{{ with index . 0 }}{{ .HostPort }}{{end}}{{end}}')
    export CMOS_HOST="localhost:$cmos_port"
    echo "# Test CMOS is running at http://$CMOS_HOST."

    local cbs_host
    cbs_host=$(echo "$COUCHBASE_SERVER_HOSTS" | head -n1)

    # Add CBS to Prometheus
    # shellcheck disable=SC2001
    curl -fsS -X POST \
      -H "Content-Type: application/json" \
      --data '{"hostname":"'"$(echo "$cbs_host" | sed -e 's/:[0-9]*$//')"'", "couchbaseConfig": {"managementPort": 8091, "username": "Administrator", "password": "password"}}' \
      "http://$CMOS_HOST/config/api/v1/clusters/add"

    wait_for_url 10 "http://$CMOS_HOST/prometheus/-/ready"

    curl -fsS -X POST "http://$CMOS_HOST/prometheus/-/reload"
}

# Starts a Couchbase cluster and CMOS container.
#
# Parameters:
# $SMOKE_NODES: The number of nodes to start (defaults to 3)
#
# This function will set the following variables with its results:
# $COUCHBASE_SERVER_HOSTS: the hostname/IP and management port of every CBS node, separated by newlines
#   (Note that they may not be accessible from localhost, e.g. if running in a container - they'll be accessible to CMOS though)
# $COUCHBASE_SERVER_NODES: the number of nodes running
# $CMOS_HOST: the hostname/IP and nginx port of the running CMOS container
#
# Note: do not call this function using BATS run! Otherwise its variables will not be set.
function start_smoke_cluster() {
    local nodes=${SMOKE_NODES:-3}
    echo "# Starting smoke cluster for platform $TEST_PLATFORM with $nodes nodes of CB $COUCHBASE_SERVER_IMAGE"
    case $TEST_PLATFORM in
        native)
            export VAGRANT_NODES=$nodes
            export COUCHBASE_SERVER_NODES=$nodes
            start_vagrant_cluster "$COUCHBASE_SERVER_VERSION" "centos7"
            while IFS= read -r host; do
              wait_for_url 10 "$host/ui"
            done <<< "$COUCHBASE_SERVER_HOSTS"
            initialize_couchbase_cluster "docker run --rm -i --network host $COUCHBASE_SERVER_IMAGE /opt/couchbase/bin/couchbase-cli"
            _start_cmos
            ;;
        containers)
            ensure_variables_set CMOS_IMAGE
            ensure_variables_set COUCHBASE_SERVER_IMAGE
            if cb_version_lt "7.0.0"; then
              # Build a new image, containing the Exporter
              DOCKER_BUILDKIT=1 docker build -t "$COUCHBASE_SERVER_IMAGE-exporter" \
               --build-arg COUCHBASE_SERVER_IMAGE="$COUCHBASE_SERVER_IMAGE" \
                -f "$RESOURCES_ROOT/containers/cb-with-exporter.Dockerfile" "$RESOURCES_ROOT/containers"
              export COUCHBASE_SERVER_IMAGE="$COUCHBASE_SERVER_IMAGE-exporter"
            fi
            # We're creating these manually instead of using Compose because we need to support a variable number of nodes.
            docker network create cmos_test
            for i in $(seq 1 "$nodes"); do
                local extra_args=""
                if [ "$i" -eq 1 ]; then
                    extra_args="-p 8091"
                fi
                # shellcheck disable=SC2086
                docker run --rm -d --name "test_couchbase$i" --network cmos_test --network-alias="couchbase$i.local" \
                  $extra_args "$COUCHBASE_SERVER_IMAGE"
            done
            COUCHBASE_SERVER_HOSTS=$(seq -f "couchbase%g.local" 1 "$nodes")
            export COUCHBASE_SERVER_HOSTS
            export COUCHBASE_SERVER_NODES=$nodes
            # Can't just use COUCHBASE_SERVER_HOSTS as they won't be accessible outside the container network
            local mgmt_port
            mgmt_port=$(docker inspect test_couchbase1 -f '{{with index .NetworkSettings.Ports "8091/tcp"}}{{ with index . 0 }}{{ .HostPort }}{{end}}{{end}}')
            wait_for_url 10 "http://localhost:$mgmt_port/ui"
            initialize_couchbase_cluster "docker run --rm -i --network cmos_test $COUCHBASE_SERVER_IMAGE /opt/couchbase/bin/couchbase-cli"
            _start_cmos --network=cmos_test
            ;;
        kubernetes)
            echo "TODO" # CMOS-97
            ;;
    esac
}

# Tears down the setup from start_smoke_cluster.
#
# Parameters:
# $SMOKE_NODES: The number of nodes that were started (defaults to 3)
function teardown_smoke_cluster() {
    if [ "${SKIP_TEARDOWN:-}" == "true" ]; then
      echo "# Skipping teardown"
      return 0
    fi
    local nodes=${SMOKE_NODES:-3}
    echo "# Tearing down smoke cluster for platform $TEST_PLATFORM with $nodes nodes"
    case $TEST_PLATFORM in
        native)
            docker stop cmos
            export VAGRANT_NODES=$nodes
            # TODO: move this into the matrix
            teardown_vagrant_cluster "$COUCHBASE_SERVER_VERSION" "centos7"
            ;;
        containers)
            docker stop cmos
            for i in $(seq 1 "$nodes"); do
                docker stop "test_couchbase$i"
            done
            docker network rm cmos_test
            ;;
        kubernetes)
            echo "TODO" # CMOS-97
            ;;
    esac
}
