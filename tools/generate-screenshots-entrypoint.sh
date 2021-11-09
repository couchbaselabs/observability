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

# Custom entrypoint for prometheus to first extract the snapshot(s) we provide then launch.
set -eu

PROMETHEUS_SNAPSHOT_DIR=${PROMETHEUS_SNAPSHOT_DIR:-/data_snapshot}
PROMETHEUS_STORAGE_PATH=${PROMETHEUS_STORAGE_PATH-/prometheus/data/}

for SNAPSHOT_FILE in "${PROMETHEUS_SNAPSHOT_DIR}"/*.zip; do
    if [[ ! -r "$SNAPSHOT_FILE" ]]; then
        echo "Unable to extract $SNAPSHOT_FILE"
        continue
    fi
    echo "Extracting snapshot $SNAPSHOT_FILE"
    unzip "$SNAPSHOT_FILE" -d "$PROMETHEUS_STORAGE_PATH"
done

# The assumption is this is run with the original Prometheus entrypoint disabled.
# We therefore then invoke the original at the end.
exec /entrypoints/prometheus.sh
