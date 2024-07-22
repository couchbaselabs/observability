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
    if ! jq -e 'has("__requires")' "$source" > /dev/null; then
      printf "\tFAIL: not exported using 'share for external use'\n"
      exit_code=1
    fi

    # Check that all panels have defined datasources - which one Grafana considers to be the "default" is unpredictable
    if ! jq -e '.panels | map(select(.type != "row" and .type != "text")) | map(.datasource) | all(. != null)' "$source" >/dev/null; then
      printf "\tFAIL: panels missing 'datasource': "
      jq -c '.panels | map(select(.type != "row" and .type != "text")) | map(select(.datasource == null)) | map(.title)' "$source"
    fi

    # Now, check over all panels and template variables to find the data sources they use,
    # and ensure that they themselves are variables
    panel_ds_vars=$(jq -cer '.panels | map(select(.type != "row")) | map(.datasource.uid) | unique | .[] | select(. != null and . != "-- Mixed --") | sub("\\$\\{(?<name>.*)\\}"; "\(.name)")' "$source")
    template_ds_vars=$(jq -cr '.templating.list | map(select(.type == "query")) | map(if (.datasource | type) == "object" then .datasource.uid else .datasource end) | unique | .[] | sub("\\$\\{(?<name>.*)\\}"; "\(.name)")' "$source")
    while IFS= read -r var; do
      if ! jq -e '.templating.list[] | select(.type == "datasource" and .name == "'"$var"'")' "$source" >/dev/null; then
        printf "\tFAIL: data source %s not defined as a variable\n" "$var"
      fi
    done < <(printf "%s\n%s" "$panel_ds_vars" "$template_ds_vars" | sort -u)
done < <(find "$DASHBOARDS_PATH" -type f -name '*.json' -print0)

exit "$exit_code"
