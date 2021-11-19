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
CMOS_LOGS=${CMOS_LOGS:-/logs}

FLUENT_BIT_CONFIG_FILE=${FLUENT_BIT_CONFIG_FILE:-/etc/fluent-bit/fluent-bit.conf}
FLUENT_BIT_HTTP_PORT=${FLUENT_BIT_HTTP_PORT:-2020}

# Set up Prometheus scraping for this target - this allows us to dynamically turn it on/off
PROMETHEUS_DYNAMIC_INTERNAL_DIR=${PROMETHEUS_DYNAMIC_INTERNAL_DIR:-/etc/prometheus/couchbase/monitoring/}
mkdir -p "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"
cat > "${PROMETHEUS_DYNAMIC_INTERNAL_DIR}"/fluentbit.json << __EOF__
[
    {
      "targets": [
        "localhost:$FLUENT_BIT_HTTP_PORT"
      ],
      "labels": {
        "__metrics_path__": "/api/v1/metrics/prometheus",
        "job": "fluentbit",
        "container": "monitoring"
      }
    }
]
__EOF__

/fluent-bit/bin/fluent-bit -c "${FLUENT_BIT_CONFIG_FILE}"
