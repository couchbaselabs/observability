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

cp /etc/*-release.txt "$tmpdir"

# Environment
env > "$tmpdir/env.txt"

# Running processes
ps > "$tmpdir/ps.txt"

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

# Grab various overridden paths
mkdir -p "$tmpdir/dynamic-config"
for override_var in PROMETHEUS_CONFIG_FILE PROMETHEUS_CONFIG_TEMPLATE_FILE JAEGER_CONFIG_FILE LOKI_CONFIG_FILE ALERTMANAGER_CONFIG_FILE; do
  cp "${!override_var}" "$tmpdir/dynamic-config/$(basename "${!override_var}")"
done

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

# Prometheus stats snapshot
snapshot_file_name=$(curl -sS -X POST http://localhost:9090/prometheus/api/v1/admin/tsdb/snapshot | jq -r '.data.name')
cp -r "$PROMETHEUS_STORAGE_PATH/snapshots/$snapshot_file_name" "$tmpdir/prometheus-snapshot"

# Prom/Loki dynamic endpoints
curl -sS -o "$tmpdir/grafana-frontend-settings.json" "http://localhost:3000/grafana/api/frontend/settings"

curl -sS -o "$tmpdir/loki-buildinfo.json" "http://localhost:3100/loki/api/v1/status/buildinfo"
curl -sS -o "$tmpdir/loki-config.yml" "http://localhost:3100/config"

curl -sS -o "$tmpdir/prom-buildinfo.json" "http://localhost:9090/prometheus/api/v1/status/buildinfo"
curl -sS -o "$tmpdir/prom-runtimeinfo.json" "http://localhost:9090/prometheus/api/v1/status/runtimeinfo"
curl -sS -o "$tmpdir/prom-flags.json" "http://localhost:9090/prometheus/api/v1/status/flags"
curl -sS -o "$tmpdir/prom-tsdb-status.json" "http://localhost:9090/prometheus/api/v1/status/tsdb"

curl -sS -o "$tmpdir/prom-config.json" "http://localhost:9090/prometheus/api/v1/status/config"
curl -sS -o "$tmpdir/prom-targets.json" "http://localhost:9090/prometheus/api/v1/targets"

# Important Prometheus series
curl -sS -o "$tmpdir/prom-series.json" "http://localhost:9090/prometheus/api/v1/series?match[]=multimanager_cluster_checker_status&match[]=multimanager_node_checker_status&match[]=multimanager_bucket_checker_status&match[]=cm_rest_request_enters_total&match[]=cbnode_up"

# These ones use `curl -v` instead, because the actual endpoints don't give us much info
curl -sv "http://localhost:3100/ready" > "$tmpdir/loki-health.txt" 2>&1
curl -sv "http://localhost:9090/prometheus/-/healthy" > "$tmpdir/prom-health.txt" 2>&1
curl -sv "http://localhost:9093/alertmanager/-/healthy" > "$tmpdir/am-health.txt" 2>&1
curl -sv "http://localhost:14269" > "$tmpdir/jaeger-health.txt" 2>&1
curl -sv "http://localhost:3000/grafana/api/health" > "$tmpdir/grafana-health.txt" 2>&1
curl -sv "http://localhost:8080/_meta/status" > "$tmpdir/nginx-status.txt" 2>&1

# Cluster Monitor endpoints
if [ -f "/bin/cbmultimanager" ]; then
  curl -sS -u "$CB_MULTI_ADMIN_USER:$CB_MULTI_ADMIN_PASSWORD" -o "$tmpdir/couchbase-cluster-monitor-self.json" "http://localhost:7196/api/v1/self"
  curl -sS -u "$CB_MULTI_ADMIN_USER:$CB_MULTI_ADMIN_PASSWORD" -o "$tmpdir/couchbase-clusters.json" "http://localhost:7196/api/v1/clusters"
  curl -sS -u "$CB_MULTI_ADMIN_USER:$CB_MULTI_ADMIN_PASSWORD" -o "$tmpdir/couchbase-checkers.json" "http://localhost:7196/api/v1/checkers"
  curl -sS -u "$CB_MULTI_ADMIN_USER:$CB_MULTI_ADMIN_PASSWORD" -o "$tmpdir/couchbase-dismissals.json" "http://localhost:7196/api/v1/dismissals"
else
  touch "$tmpdir/no-cluster-monitor"
fi

# Copy over all logs
# Do this at the end, so anything logged because of what we do is captured
mkdir -p "$tmpdir/logs"
cp /logs/* "$tmpdir/logs/"

# Do not copy /var/log/nginx/*, because they get mapped to stdout/stderr and cp will hang forever
# shellcheck disable=SC2043
for var_log in grafana; do
  for f in /var/log/"$var_log"/*; do cp "$f" "$tmpdir/logs/$var_log.$(basename "$f")"; done
done

# Tar it up and copy it to /support
output="/tmp/support/cmosinfo-$(date -u +"%Y-%m-%dT%H-%M-%SZ").tar"
tar -cvf "$output" -C "$tmpdir" .
tar_exitcode=$?

set +x

if [ "$tar_exitcode" -eq 0 ]; then
  echo "Collected support information at $output."
  echo "If the CMOS web server is enabled, it can also be downloaded from http://<cmos-host>:8080/support/$(basename "$output")."
  echo
  echo "!!! WARNING !!!"
  echo "Currently, NO REDACTION is performed on the collected files."
  echo "We recommend you inspect them and remove any sensitive information before sending to Couchbase Support."
else
  echo "An error occurred and the diagnostics archive could not be collected."
  echo "Please inspect the output above for details."
fi