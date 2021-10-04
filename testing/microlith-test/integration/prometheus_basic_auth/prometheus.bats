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

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

setup() {
    if [ "$TEST_NATIVE" != "true" ]; then
        skip "Skipping native prometheus tests"
    fi
}

teardown() {
    if [ "$TEST_NATIVE" == "true" ]; then
        docker-compose --project-directory="${TEST_ROOT}/integration/prometheus_basic_auth" rm -v --force --stop
    fi
}

waitForRemote() {
    URL=$1
    MAX_ATTEMPTS=$2
    CREDENTIALS=$3
    ATTEMPTS=0
    until curl -s -o /dev/null "${CREDENTIALS}" "${URL}"; do
        # shellcheck disable=SC2086
        if [ $ATTEMPTS -gt $MAX_ATTEMPTS ]; then
            assert_failure "unable to communicate with $URL"
        fi
        ((ATTEMPTS++))
        sleep 10
    done
    run curl -s "${CREDENTIALS}" "${URL}"
    assert_success
}

@test "Verify pre-requisites" {
    run : "${TEST_ROOT?"Need to set TEST_ROOT"}"
    assert_success
    run : "${CMOS_PORT?"Need to set CMOS_PORT"}"
    assert_success
}

@test "Verify that basic auth can be passed by environment variable" {
    docker-compose --project-directory="${TEST_ROOT}/integration/prometheus_basic_auth" up -d --force-recreate --remove-orphans
    # Wait for Couchbase to initialise
    waitForRemote "http://localhost:8091/pools/default" 30 "-u Administrator:newpassword"
    run curl -s -u Administrator:newpassword "http://localhost:8091/pools/default"
    assert_success
    # Sometimes it isn't quite ready even after that starts 200ing
    sleep 10
    # And Prometheus, just in case
    waitForRemote "http://localhost:${CMOS_PORT}/prometheus/-/ready" 12
    # Create a user
    run docker-compose --project-directory="${TEST_ROOT}/integration/prometheus_basic_auth" exec cb1 /opt/couchbase/bin/couchbase-cli user-manage -c localhost -u Administrator -p newpassword --set --auth-domain "local" --rbac-username prometheus --rbac-password prometheus --roles external_stats_reader
    assert_success
    # Wait the length of one scrape_interval, plus some margin
    sleep 35
    # Check that we're able to scrape CB in CMOS
    run bash -c "curl -s http://localhost:${CMOS_PORT}/prometheus/api/v1/targets 2>/dev/null | jq -r '.data.activeTargets[] | select(.labels.job == "'"'"couchbase-server"'"'") | .health'"
    assert_line "up"
}
