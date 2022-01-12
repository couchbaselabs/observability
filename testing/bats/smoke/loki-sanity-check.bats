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

@test "Loki alerting rules operational" {
    # Wait for Loki to come up, write a line to it, then ping Alertmanager until the alert shows up
    # Can't use /ready because the reverse proxy will mangle the path to /loki/ready
    wait_for_url 10 "$CMOS_HOST/loki/api/v1/status/buildinfo"
    wait_for_url 10 "$CMOS_HOST/alertmanager/-/ready"

    run curl -X POST -H "Content-Type: application/json" "$CMOS_HOST/loki/api/v1/push" --data @- <<EOF
{
    "streams": [
        {
            "stream": {
                "job": "cmos-testing"
            },
            "values": [
                ["$(date +%s)000000000", "test"]
            ]
        }
    ]
}
EOF
    assert_success

    attempt=0
    while true; do
        run curl -o "$BATS_TEST_TMPDIR/alerts.json" "$CMOS_HOST/alertmanager/api/v2/alerts"
        assert_success

        run jq -c '.[] | select(.labels.alertname == "SmokeTest")' "$BATS_TEST_TMPDIR/alerts.json"
        assert_success
        if [[ "$output" == "" ]]; then
            if [ "$attempt" -lt 30 ]; then
                attempt=$(( attempt + 1 ))
                sleep 5
            else
                fail "Didn't find the smoke test alert even after $attempt attempts"
            fi
        else
            break
        fi
    done
}
