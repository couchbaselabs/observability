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

# This script env-substitutes all Loki alerting rules, then copies them to the right place.
# We do not do any more complex overriding or pre-processing than that.

if [[ -v "${DISABLE_LOKI_ALERTS_PREPARE}" ]]; then
  echo "loki_alerts_prepare.sh: disabled, not using preset alerting rules."
  exit 0
fi

set -e

if [ "${LOKI_ALERTS_INCLUDE_TEST:-}" != 'true' ]; then
  rm /etc/loki/alerting/couchbase/test-rules.yaml || true
fi

# Work on the rules, we substitute in-place to keep it simple
while IFS= read -r -d '' FILE
do
  if mv -f "${FILE}" "${FILE}".orig; then
    # We need to make sure we only substitute defined variables otherwise we remove label/annotation processing as well
    # e.g. `description: {{ $labels.node }} has condition VALUE = {{ $value }} LABELS = {{ $labels }}`
    # Using envsubst on its own would mean the $labeles and $values fields are blank
    # Therefore we pass envsubst a list of all values defined in the environment as the "only" things to substitute
    envsubst "$(env | cut -d= -f1 | sed -e 's/^/$/')" < "${FILE}".orig > "${FILE}"
    if diff -aq "${FILE}".orig "${FILE}"; then
      echo "Processed ${FILE}:"
      diff -a "${FILE}".orig "${FILE}"
    else
      rm -f "${FILE}".orig
    fi
  else
    echo "Unable to substitute any values in ${FILE} - likely read-only due to being mounted in"
  fi
done < <(find "/etc/loki/alerting/" -type f \( -name '*.yaml' -o -name '*.yml' \) -print0)

mkdir -p /etc/loki/rules/fake
shopt -s nullglob
cp /etc/loki/alerting/couchbase/*.yaml /etc/loki/alerting/custom/*.yaml /etc/loki/rules/fake/
