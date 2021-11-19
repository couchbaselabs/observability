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

ensure_variables_set BATS_SUPPORT_ROOT BATS_ASSERT_ROOT

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"

@test "verify promtail functions if nothing else is running" {
    # https://grafana.com/docs/loki/latest/clients/promtail/troubleshooting/
    echo "test log line" | promtail --stdin --dry-run --inspect --client.url http://127.0.0.1:3100/loki/api/v1/push
}

@test "verify logs are being read" {
    local promtail_url="localhost:9080"
    # Are we ready?
    wait_for_url 10 "$promtail_url/ready"

    # Are we consuming any logs?
    # Check the "promtail_files_active_total" metric is > 0
    run curl "$promtail_url/metrics"
    assert_success
    assert_line -p 'promtail_files_active_total'
    refute_line 'promtail_files_active_total 0'
}

@test "verify logs are being sent to Loki" {
    if [ -v "${DISABLE_LOKI}"; then
        skip "Loki disabled"
    fi
    local promtail_url="localhost:9080"
    # Are we ready?
    wait_for_url 10 "$promtail_url/ready"

    # Are we forwarding logs to Loki ok?
    run curl "$promtail_url/metrics"
    assert_success
    assert_line -p 'promtail_sent_bytes_total'
    refute_line 'promtail_sent_bytes_total{host="localhost:3100"} 0'
}