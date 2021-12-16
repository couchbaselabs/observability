#!/bin/bash
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

# Helper script to lint Prometheus and Loki alerting rules to ensure they conform to our standards.
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if ! command -v "yq" &> /dev/null; then
    echo "yq not installed. Install yq before running this script: https://github.com/kislyuk/yq"
    exit 1
fi

# NOTE: this doesn't check other rules (e.g. prometheus-self-monitoring), as they don't conform to our standards.
rule_files="$SCRIPT_DIR/../microlith/prometheus/alerting/couchbase/couchbase-rules.yaml $SCRIPT_DIR/../microlith/loki/alerting/couchbase/couchbase-rules.yaml"
exit_code=0

for source in $rule_files; do
    echo "Rules lint: ${source##"$SCRIPT_DIR/.."}"

    # Check that the required labels are defined
    if ! yq -e '.groups[].rules[] | [.labels.job, .labels.kind, .labels.health_check_id, .labels.health_check_name, .labels.severity] | map(type) | all(. == "string")' "$source" > /dev/null; then
      printf "\tFAIL: missing labels\n"
      exit_code=1
    fi

    # Check all defined checkers are documented
    # First extract their IDs, then search our asciidoc for them
    while IFS= read -r id; do
        if ! grep -q "$id" "$SCRIPT_DIR/../docs/modules/ROOT/partials/cmos-health-checks.adoc"; then
            printf "\tFAIL: undocumented checker %s\n" "$id"
      exit_code=1
        fi
    done <<< "$(yq -r '.groups[].rules[].labels.health_check_id | select(. | type == "string")' "$source")"
done

exit "$exit_code"

