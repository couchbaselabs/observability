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

ensure_variables_set CMOS_HOST BATS_SUPPORT_ROOT BATS_ASSERT_ROOT

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"

function metricGreaterThanZero() {
    local attempt=0
    local metric=$1
    while true; do
        run curl -o "$BATS_TEST_TMPDIR/output.json" -X GET "$CMOS_HOST/prometheus/api/v1/query?query=$metric>0"
        assert_success

        run jq -c '.data.result[] | select(.metric.job == "promtail")' "$BATS_TEST_TMPDIR/targets.json"
        assert_success
        if [[ "$output" == "" ]]; then
            if [ "$attempt" -lt 10 ]; then
                attempt=$(( attempt + 1 ))
                sleep 5
            else
                fail "$metric stayed at zero even after $attempt attempts"
            fi
        else
            break
        fi
    done
}

@test "verify logs are being ingested by Promtail and Loki" {
    # Are we ready?
    wait_for_url 10 "$CMOS_HOST/prometheus/-/ready"

    # Are we consuming any logs?
    metricGreaterThanZero promtail_files_active_total

    # Are we forwarding logs to Loki ok?
    metricGreaterThanZero promtail_sent_bytes_total
}