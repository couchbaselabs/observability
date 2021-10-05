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

# Simple script to run all native tests.
# It relies on BATS being installed, see tools/install-bats.sh

set -xueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
BATS_FORMATTER=${BATS_FORMATTER:-tap}
DOCKER_USER=${DOCKER_USER:-couchbase}
DOCKER_TAG=${DOCKER_TAG:-v1}

export BATS_ROOT=${BATS_ROOT:-$SCRIPT_DIR/../../tools/bats}
export BATS_FILE_ROOT=${BATS_FILE_ROOT:-BATS_ROOT/lib/bats-file}
export BATS_SUPPORT_ROOT=${BATS_SUPPORT_ROOT:-BATS_ROOT/lib/bats-support}
export BATS_ASSERT_ROOT=${BATS_ASSERT_ROOT:-BATS_ROOT/lib/bats-assert}
export BATS_DETIK_ROOT=${BATS_DETIK_ROOT:-BATS_ROOT/lib/bats-detik}

export TEST_NATIVE=true
export TEST_ROOT="${SCRIPT_DIR}/../microlith-test/"
export HELPERS_ROOT="${SCRIPT_DIR}/../helpers"
export CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
export CMOS_PORT=${CMOS_PORT:-8080}
# TODO: this is required for the role used by the basic auth test, this needs updating to be conditional and use the exporter
export COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:6.6.3}
bats --formatter "${BATS_FORMATTER}" --recursive "${TEST_ROOT}" --timing
