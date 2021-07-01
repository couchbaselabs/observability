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

set -ex
# For envsubst we have to export
export CLUSTER_MONITOR_USER=${CLUSTER_MONITOR_USER:-admin}
export CLUSTER_MONITOR_PWD=${CLUSTER_MONITOR_PWD:-password}

# To customise the Prometheus configuration used, set these values at launch
PROMETHEUS_CONFIG_FILE=${PROMETHEUS_CONFIG_FILE:-/etc/prometheus/prometheus-runtime.yml}
PROMETHEUS_CONFIG_TEMPLATE_FILE=${PROMETHEUS_CONFIG_TEMPLATE_FILE:-/etc/prometheus/prometheus-template.yml}
PROMETHEUS_URL_SUBPATH=${PROMETHEUS_URL_SUBPATH-/prometheus/}
PROMETHEUS_STORAGE_PATH=${PROMETHEUS_STORAGE_PATH-/prometheus}

# Example variables to tune with - it would be nicer to include defaults in the file but envsubst does not support that:
export COUCHBASE_ACTIVE_RESIDENT_RATIO_ALERT_THRESHOLD=${COUCHBASE_ACTIVE_RESIDENT_RATIO_ALERT_THRESHOLD:-100}
export COUCHBASE_ACTIVE_RESIDENT_RATIO_ALERT_DURATION=${COUCHBASE_ACTIVE_RESIDENT_RATIO_ALERT_DURATION:-1m}

set +x

# Substitute environment variables as Prometheus does not support this (actively refused to do so)
# https://www.robustperception.io/environment-substitution-with-docker
if [[ -f "${PROMETHEUS_CONFIG_TEMPLATE_FILE}" ]] ; then
  # Make sure to leave alone anything that is not a defined environment variable
  envsubst "$(env | cut -d= -f1 | sed -e 's/^/$/')"  < "${PROMETHEUS_CONFIG_TEMPLATE_FILE}" > "${PROMETHEUS_CONFIG_FILE}"
fi

# Now work on the rules, we substitute in-place to keep it simple
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
    echo "Unable to substitue any values in ${FILE} - likely read-only due to being mounted in"
  fi
done < <(find "/etc/prometheus/alerting" -type f \( -name '*.yaml' -o -name '*.yml' \) -print0)

# From: https://prometheus.io/docs/prometheus/latest/configuration/configuration/
# A configuration reload is triggered by sending a SIGHUP to the Prometheus process or
# sending a HTTP POST request to the /-/reload endpoint.

/bin/prometheus --config.file="${PROMETHEUS_CONFIG_FILE}" \
                --storage.tsdb.path="${PROMETHEUS_STORAGE_PATH}" \
                --web.console.libraries=/usr/share/prometheus/console_libraries \
                --web.console.templates=/usr/share/prometheus/consoles \
                --web.external-url="${PROMETHEUS_URL_SUBPATH}" \
                --web.enable-lifecycle

# https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus