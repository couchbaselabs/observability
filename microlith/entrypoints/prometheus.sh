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
export CB_MULTI_ADMIN_USER=${CB_MULTI_ADMIN_USER:-admin}
export CB_MULTI_ADMIN_PASSWORD=${CB_MULTI_ADMIN_PASSWORD:-password}
export CB_SERVER_AUTH_USER=${CB_SERVER_AUTH_USER:-Administrator}
export CB_SERVER_AUTH_PASSWORD=${CB_SERVER_AUTH_PASSWORD:-password}

# To customise the Prometheus configuration used, set these values at launch
PROMETHEUS_CONFIG_FILE=${PROMETHEUS_CONFIG_FILE:-/etc/prometheus/prometheus-runtime.yml}
PROMETHEUS_CONFIG_TEMPLATE_FILE=${PROMETHEUS_CONFIG_TEMPLATE_FILE:-/etc/prometheus/prometheus-template.yml}
PROMETHEUS_URL_SUBPATH=${PROMETHEUS_URL_SUBPATH-/prometheus/}
PROMETHEUS_STORAGE_PATH=${PROMETHEUS_STORAGE_PATH-/prometheus/data/}
PROMETHEUS_STORAGE_MAX_SIZE=${PROMETHEUS_STORAGE_MAX_SIZE:-512MB}
PROMETHEUS_RETENTION_TIME=${PROMETHEUS_RETENTION_TIME:-15d}

# Promtheus is a bit funny about it's CLI flags, you cannot have empty space apparently
PROMETHEUS_EXTRA_ARGS=${PROMETHEUS_EXTRA_ARGS:---}

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

# Prepare the alerting rules (the script itself handles the disable variable)
bash /etc/prometheus/scripts/alerts_prepare.sh

# From: https://prometheus.io/docs/prometheus/latest/configuration/configuration/
# A configuration reload is triggered by sending a SIGHUP to the Prometheus process or
# sending a HTTP POST request to the /-/reload endpoint.
/bin/prometheus --config.file="${PROMETHEUS_CONFIG_FILE}" \
                --enable-feature=memory-snapshot-on-shutdown \
                --storage.tsdb.path="${PROMETHEUS_STORAGE_PATH}" \
                --storage.tsdb.retention.size="${PROMETHEUS_STORAGE_MAX_SIZE}" \
                --storage.tsdb.retention.time="${PROMETHEUS_RETENTION_TIME}" \
                --web.console.libraries=/usr/share/prometheus/console_libraries \
                --web.console.templates=/usr/share/prometheus/consoles \
                --web.external-url="${PROMETHEUS_URL_SUBPATH}" \
                --web.enable-lifecycle "${PROMETHEUS_EXTRA_ARGS}"

# https://www.robustperception.io/using-external-urls-and-proxies-with-prometheus
