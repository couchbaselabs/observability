#!/usr/bin/env bash
set -euo pipefail

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

# Install the Grafana plugins given in GF_INSTALL_PLUGINS.
# Follows the exact same format as expected by the official Grafana Docker image - a comma-separated list of plugins,
# optionally with the version, space-separated. For example, `marcusolsson-json-datasource 1.3.0,other-test-plugin`.
# Note that the ZIP URL syntax is currently not supported, so only plugins from grafana.com can be installed.
# Upstream docs: https://grafana.com/docs/grafana/latest/installation/docker/#install-official-and-community-grafana-plugins

if [ -n "$GF_INSTALL_PLUGINS" ]; then
  old_ifs=$IFS
  IFS=','
  tmp_dir=$(mktemp -d)
  for plugin_arg in ${GF_INSTALL_PLUGINS}; do
    IFS=$old_ifs
    # Can be just the bare name (for the latest version) or `name version`
    if [[ "$plugin_arg" =~ ^(.*)[[:space:]](.*)$ ]]; then
      plugin_name="${BASH_REMATCH[1]}"
      plugin_version="${BASH_REMATCH[2]}"
    else
      plugin_name=$plugin_arg
      plugin_version=latest
    fi
    wget -O "$tmp_dir/$plugin_name.zip" "https://grafana.com/api/plugins/$plugin_name/versions/$plugin_version/download"
    unzip -d "$GF_PATHS_PLUGINS" "$tmp_dir/$plugin_name.zip"
    rm "$tmp_dir/$plugin_name.zip"
  done
fi
