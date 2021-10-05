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

load "$HELPERS_ROOT/test-helpers.bash"

ensure_variables_set TEST_ROOT CMOS_PORT BATS_SUPPORT_ROOT BATS_ASSERT_ROOT BATS_FILE_ROOT HELPERS_ROOT COUCHBASE_SERVER_IMAGE

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"
<<<<<<< HEAD:testing/bats/integration/containers/prometheus_basic_auth/prometheus.bats

setup() {
    echo "Verify pre-requisites"
    run : "${TEST_ROOT?"Need to set TEST_ROOT"}"
    assert_success
    run : "${CMOS_PORT?"Need to set CMOS_PORT"}"
    assert_success
}
=======
load "$HELPERS_ROOT/couchbase-helpers.bash"
load "$HELPERS_ROOT/url-helpers.bash"
>>>>>>> origin/main:testing/bats/integration/prometheus_basic_auth/prometheus_basic_auth.bats

teardown() {
    if [ "$SKIP_TEARDOWN" == "true" ]; then
        skip "Skipping teardown"
    elif [ "$TEST_NATIVE" == "true" ]; then
        run docker-compose --project-directory="${BATS_TEST_DIRNAME}" logs --timestamps || echo "Unable to get compose output"
        run docker-compose --project-directory="${BATS_TEST_DIRNAME}" rm -v --force --stop
    fi
}

@test "Verify that basic auth can be passed by environment variable" {
    # shellcheck disable=SC2076
    if [[ ! "$COUCHBASE_SERVER_IMAGE" =~ "7." ]]; then
        skip "Skipping, only applicable to Server 7.x"
    fi
    docker-compose --project-directory="${BATS_TEST_DIRNAME}" up -d --force-recreate --remove-orphans
    # Wait for Couchbase to initialise
    wait_for_curl 30 "http://localhost:8091/pools/default" -u Administrator:newpassword
    # Sometimes it isn't quite ready even after that starts 200ing
    sleep 10
    # And Prometheus, just in case
    wait_for_url 12 "http://localhost:${CMOS_PORT}/prometheus/-/ready"
    # Create a user
    run docker-compose --project-directory="${BATS_TEST_DIRNAME}" exec -T cb1 /opt/couchbase/bin/couchbase-cli user-manage -c localhost -u Administrator -p newpassword --set --auth-domain "local" --rbac-username prometheus --rbac-password prometheus --roles external_stats_reader
    assert_success
    # Wait the length of one scrape_interval, plus some margin
    sleep 35
    # Check that we're able to scrape CB in CMOS
    run bash -c "curl -s http://localhost:${CMOS_PORT}/prometheus/api/v1/targets 2>/dev/null | jq -r '.data.activeTargets[] | select(.labels.job == "'"'"couchbase-server"'"'") | .health'"
    assert_line "up"
}
