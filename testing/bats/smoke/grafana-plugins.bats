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

@test "community Grafana plugins present" {
    wait_for_url 10 "$CMOS_HOST/grafana/api/health"
    run curl -fs -o "$BATS_TEST_TMPDIR/output.json" "$CMOS_HOST/grafana/api/plugins"
    assert_success
    run jq -c '.[] | select(.signatureType == "community")' "$BATS_TEST_TMPDIR/output.json"
    [ -n "$output" ]
    # Add any specific plugins we user here
    assert_line -p 'marcusolsson-json-datasource'
}
