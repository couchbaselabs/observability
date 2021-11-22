#!/usr/bin/env bash

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

# Sanity checks on our Grafana dashboard files
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
DASHBOARDS_PATH="$SCRIPT_DIR/../microlith/grafana/provisioning/dashboards"
exit_code=0

while IFS= read -r -d '' source
do
    echo "Dashboard lint: ${source##"$DASHBOARDS_PATH"}"
    # Check that all time ranges are relative, otherwise you get confusing import behaviour
    if ! jq -e '[.time.from, .time.to] | all(contains("now"))' "$source" > /dev/null; then
      printf "\tFAIL: non-relative time range\n"
      exit_code=1
    fi

    # Check that they haven't been shared for external use
    if jq -e 'has("__requires")' "$source" > /dev/null; then
      printf "\tFAIL: exported using 'share for external use'\n"
      exit_code=1
    fi
done < <(find "$DASHBOARDS_PATH" -type f -name '*.json' -print0)

exit "$exit_code"
