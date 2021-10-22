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

set -euo pipefail

export CMOS_CFG_BIN=${CMOS_CFG_BIN:-/cmoscfg}

export CMOS_CFG_PATH=${CMOS_CFG_PATH:-/etc/cmos/config.yaml}
export CMOS_CFG_HTTP_PATH_PREFIX=${CMOS_CFG_HTTP_PATH_PREFIX:-}
export CMOS_CFG_HTTP_HOST=${CMOS_CFG_HTTP_HOST:-0.0.0.0}
export CMOS_CFG_HTTP_PORT=${CMOS_CFG_HTTP_PORT:7194}

if [[ $# -gt 0 ]]; then
    echo "Running custom: $*"
    exec "$@"
else
  if [[ -x "${CMOS_CFG_BIN}" ]]; then
          # Making all parameters explicit so people can see how to configure the CLI.
          "${CMOS_CFG_BIN}" \
            -config-path "${CMOS_CFG_PATH}" \
            -http-prefix "${CMOS_CFG_HTTP_PATH_PREFIX}" \
            -http-host "${CMOS_CFG_HTTP_HOST}" \
            -http-port "${CMOS_CFG_HTTP_PORT}"
      else
          echo "ERROR: No executable to run: CMOS_CFG_BIN=${CMOS_CFG_BIN}"
      fi
fi
