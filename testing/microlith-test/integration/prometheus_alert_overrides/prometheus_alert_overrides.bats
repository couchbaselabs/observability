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
load "../helpers"

export CMOS_IMAGE=${CMOS_IMAGE:-couchbase/observability-stack:v1}

setup() {
    if [ "$TEST_INTEGRATION" == "true" ]; then
        skip "Skipping integration tests"
    fi
    find couchbase custom overrides -regex '.*\.ya?ml\.orig$' -delete
}

teardown() {
    if [ "$SKIP_TEARDOWN" == "true" ]; then
        echo "# Skipping teardown. Make sure to manually run the commands in teardown()." >&3
        return
    fi
    docker-compose rm -v --force --stop
}

@test "Alert overrides (generated YAML file)" {
    docker-compose up -d --force-recreate --remove-orphans
    run docker-compose exec cmos cat /etc/prometheus/alerting/generated/alerts.yaml
    assert_line -p 'expr: untouched'
    assert_line -p 'expr: overridden{foo!="true"}'
    assert_line -p 'expr: disabled{foo!="true"}'
    refute_line -p 'expr: disabled{}'
    assert_line -p 'expr: overridden{foo="true"}'
}

@test "Alert overrides (API JSON output)" {
    docker-compose up -d --force-recreate --remove-orphans
    sleep 5 # ensure Prom has a chance to start up

    prom_port=$(get_service_port prometheus_alert_overrides_cmos 9090)
    echo "Prometheus is on port $prom_port"

    tempdir=$(mktemp -d 2>/dev/null || mktemp -d -t 'cmos-test-prometheus_alert_overrides')
    curl --silent --show-error --fail http://localhost:"${prom_port}"/prometheus/api/v1/rules\?type=alert > "$tempdir/rules.json"
    echo "/api/v1/rules output: $(cat "$tempdir/rules.json")"

    run jq -r '.data.groups[].rules[].query' "$tempdir/rules.json"
    assert_line 'untouched'
    assert_line 'overridden{foo!="true"}'
    assert_line 'disabled{foo!="true"}'
    refute_line 'disabled{}'
    assert_line 'overridden{foo="true"}'
    assert_line 'custom'
    rm -r "$tempdir"
}
