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

# Don't set euo pipefail - we want this script to be as resilient as possible
set -x

echo "Starting collect-information.sh..."

tmpdir=${TEMPORARY_DIRECTORY:-$(mktemp -d)}
exec &> >(tee -a "$tmpdir/collect-information.sh.log")

cp /etc/couchbase-cluster-monitor-release.txt "$tmpdir"
cp /etc/cmos-release.txt "$tmpdir"

# Environment
env > "$tmpdir/env.txt"

# Copy over all logs
mkdir -p "$tmpdir/logs"
cp /logs/* "$tmpdir/logs/"

# Do not copy /var/log/nginx/*, because they get mapped to stdout/stderr and cp will hang forever
# shellcheck disable=SC2043
for var_log in grafana; do
  for f in /var/log/"$var_log"/*; do cp "$f" "$tmpdir/logs/$var_log.$(basename "$f")"; done
done

# Configuration
mkdir -p "$tmpdir/config"

while IFS= read -r dir; do
  cp -r "$dir" "$tmpdir/config/$(basename "$dir")/"
done <<EOF
/etc/alertmanager/
/etc/prometheus/
/etc/grafana/
/etc/jaeger/
/etc/loki/
/etc/nginx/
EOF

# Entry points
mkdir -p "$tmpdir/entrypoints"
cp /entrypoints/* "$tmpdir/entrypoints/"

# Misc scripts
mkdir -p "$tmpdir/scripts"
cp -r /scripts "$tmpdir/scripts/"
cp /run.sh "$tmpdir/scripts/"
cp /collect-information.sh "$tmpdir/scripts/"

# Grafana plugins
mkdir -p "$tmpdir/grafana-plugins"
for d in /var/lib/grafana/plugins/*; do
  cp "$d/plugin.json" "$tmpdir/grafana-plugins/$(basename "$d").json"
done

# Tar it up and copy it to /support
output="/tmp/support/cmosinfo-$(date -u +"%Y-%m-%dT%H:%M:%SZ").tar"
# shellcheck disable=SC2164
pushd "$tmpdir"
  tar cvf "$output" ./*
# shellcheck disable=SC2164
popd

set +x

echo "Collected support information at $output."
echo "If the CMOS web server is enabled, it can also be downloaded from http://<cmos-host>:8080/support/$(basename "$output")."
echo
echo "!!! WARNING !!!"
echo "Currently, NO REDACTION is performed on the collected files."
echo "We recommend you inspect them and remove any sensitive information before sending to Couchbase Support."
