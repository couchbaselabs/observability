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

# Helper function to wrap querying Prometheus and processing the result.
# Parameters:
# $1: the number of nodes we expect to see
# Returns:
# 0 if we get the expected number of nodes
# 1 if we got a result, but it didn't have the expected number of nodes
# 2 if we got nothing at all, or something else went wrong
function try_query() {
  local expected_nodes=$1
  run curl -o "$BATS_TEST_TMPDIR/output.json" -X GET "$CMOS_HOST/prometheus/api/v1/query?query=cbnode_up==1"
  if [ "$status" -ne 0 ]; then
    return 2
  fi

  run jq -c '.data.result[]' "$BATS_TEST_TMPDIR/output.json"
  if [ "$status" -ne 0 ]; then
    return 2
  fi

  if [ "${#lines[@]}" -eq "$expected_nodes" ]; then
    return 0
  elif [ "${#lines[@]}" -gt 0 ]; then
    return 1
  else
    return 2
  fi
}

@test "Couchbase Exporter is scraped" {
  wait_for_url 10 "$CMOS_HOST/prometheus/-/ready"

  local attempt=0
  local max_attempts=10
  local failure_reason_code
  while [ "$attempt" -lt "$max_attempts" ]; do
    if try_query "$COUCHBASE_SERVER_NODES"; then
      break
    else
      failure_reason_code=$?
      attempt=$(( attempt + 1 ))
      sleep 5
      continue
    fi
  done
  if [ "$attempt" -eq "$max_attempts" ]; then
    local failure_message
    case $failure_reason_code in
      1)
        failure_message="cbnode_up did not have the expected number of results after $attempt attempts."
        ;;
      2)
        failure_message="Failed to query cbnode_up after $attempt attempts."
        ;;
      *)
        failure_message="try_query returned $failure_reason_code after $attempt attempts."
        ;;
    esac
    fail "$failure_message Last query result: $(cat "$BATS_TEST_TMPDIR/output.json")"
  fi
}
