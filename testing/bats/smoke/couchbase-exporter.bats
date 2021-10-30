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

function try_query() {
  run curl -o "$BATS_TEST_TMPDIR/output.json" -X GET "$CMOS_HOST/prometheus/api/v1/query?query=cbnode_up==1"
  if [ "$status" -ne 0 ]; then
    return 1
  fi

  run jq -c '.data.result[]' "$BATS_TEST_TMPDIR/output.json"
  if [ "$status" -ne 0 ]; then
    return 1
  fi

  if [ "${#lines[@]}" -gt 0 ]; then
    return 0
  else
    return 1
  fi
}

@test "Couchbase Exporter is scraped" {
  wait_for_url 10 "$CMOS_HOST/prometheus/-/ready"

  attempt=0
  max_attempts=10
  while [ "$attempt" -lt "$max_attempts" ]; do
    if try_query; then
      break
    else
      attempt=$(( attempt + 1 ))
      sleep 5
      continue
    fi
  done
  if [ "$attempt" -eq "$max_attempts" ]; then
    fail "Failed to query cbnode_up after $attempt attempts. Last query result: $(cat "$BATS_TEST_TMPDIR/output.json")"
  fi
}
