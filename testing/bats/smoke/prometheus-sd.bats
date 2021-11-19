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

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"

@test "Prometheus finds Couchbase Server targets" {
    wait_for_url 10 "$CMOS_HOST/prometheus/-/ready"

    attempt=0
    while true; do
        run curl -o "$BATS_TEST_TMPDIR/targets.json" "$CMOS_HOST/prometheus/api/v1/targets"
        assert_success

        run jq -c '.data.activeTargets[] | select(.labels.job | contains("couchbase-server"))' "$BATS_TEST_TMPDIR/targets.json"
        assert_success
        if [[ "$output" == "" ]]; then
            if [ "$attempt" -lt 10 ]; then
                attempt=$(( attempt + 1 ))
                sleep 5
            else
                fail "Didn't find any targets even after $attempt attempts"
            fi
        else
            break
        fi
    done
}
