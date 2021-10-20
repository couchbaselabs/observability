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
load "$HELPERS_ROOT/url-helpers.bash"

ensure_variables_set CMOS_HOST BATS_SUPPORT_ROOT BATS_ASSERT_ROOT

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"

@test "nginx proxies Prometheus correctly" {
    wait_for_url 10 "$CMOS_HOST/prometheus/-/ready"
    run curl -fs -o "$BATS_TEST_TMPDIR/output.json" "$CMOS_HOST/prometheus/api/v1/status/buildinfo"
    assert_success
    run jq -c . "$BATS_TEST_TMPDIR/output.json"
    assert_output -p '"status":"success"'
}

@test "nginx proxies Grafana correctly" {
    wait_for_url 10 "$CMOS_HOST/grafana/api/health"
    run curl -fs -o "$BATS_TEST_TMPDIR/output.json" "$CMOS_HOST/grafana/api/health"
    assert_success
    run jq -c . "$BATS_TEST_TMPDIR/output.json"
    assert_output -p '"database":"ok"'
}

@test "nginx proxies Alertmanager correctly" {
    wait_for_url 10 "$CMOS_HOST/alertmanager/-/ready"
    run curl -fs -o "$BATS_TEST_TMPDIR/output.json" "$CMOS_HOST/alertmanager/-/ready"
    assert_success
}
