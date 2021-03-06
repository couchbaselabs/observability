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

set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

if [[ "${SKIP_BATS:-no}" != "yes" ]]; then
    # No point shell checking it as done separately anyway
    # shellcheck disable=SC1091
    /bin/bash "${SCRIPT_DIR}/../tools/install-bats.sh"
fi

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/test-common.sh"
# Anything that is not common now specified:
export TEST_PLATFORM=native

# This function will call `exit`, so any cleanup must be done inside of it.
run_tests "${1-}"
