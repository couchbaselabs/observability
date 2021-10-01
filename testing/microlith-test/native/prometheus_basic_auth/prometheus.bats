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

load "$BATS_DETIK_ROOT/utils.bash"
load "$BATS_DETIK_ROOT/linter.bash"
load "$BATS_DETIK_ROOT/detik.bash"
load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

setup() {
    if [ "$TEST_NATIVE" == "true" ]; then
        skip "Skipping native prometheus tests"
    fi
}

teardown() {
    if [ "$TEST_NATIVE" == "true" ]; then
        docker-compose --project-directory="${TEST_ROOT}/native/prometheus_basic_auth" rm -v --force --stop
    fi
}

@test "Verify that basic auth can be passed by environment variable" {
    docker-compose --project-directory="${TEST_ROOT}/native/prometheus_basic_auth" up -d --force-recreate --remove-orphans
    # Wait for Couchbase to initialise
    while true; do
        if curl -s -o /dev/null -u Administrator:newpassword http://127.0.0.1:8091/pools/default; then
            break
        fi
    done
    # Sometimes it isn't quite ready even after that starts 200ing
    sleep 10
    # And Prometheus, just in case
    while true; do
        if curl -s -o /dev/null "http://127.0.0.1:${CMOS_PORT}/prometheus/-/ready"; then
            break
        fi
    done
    # Create a user
    docker-compose exec cb1 /opt/couchbase/bin/couchbase-cli user-manage -c localhost -u Administrator -p newpassword --set --auth-domain "local" --rbac-username prometheus --rbac-password prometheus --roles external_stats_reader
    # Wait the length of one scrape_interval, plus some margin
    sleep 35
    # Check that we're able to scrape CB
    run bash -c "curl -s http://localhost:${CMOS_PORT}/prometheus/api/v1/targets 2>/dev/null | jq -r '.data.activeTargets[] | select(.labels.job == "'"'"couchbase-server"'"'") | .health'"
    assert_line "up"
}
