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
set -eo pipefail

# Profile script for common variables
TEST_COMMON_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TEST_ROOT="${TEST_COMMON_SCRIPT_DIR}/bats/"
export HELPERS_ROOT="${TEST_COMMON_SCRIPT_DIR}/helpers/"
export RESOURCES_ROOT="${TEST_COMMON_SCRIPT_DIR}/resources/"

export DOCKER_USER=${DOCKER_USER:-couchbase}
export DOCKER_TAG=${DOCKER_TAG:-v1}
export CMOS_IMAGE=${CMOS_IMAGE:-$DOCKER_USER/observability-stack:$DOCKER_TAG}
export CMOS_PORT=${CMOS_PORT:-8080}
export COUCHBASE_SERVER_VERSION=${COUCHBASE_SERVER_VERSION:-6.6.3}
export COUCHBASE_SERVER_IMAGE=${COUCHBASE_SERVER_IMAGE:-couchbase/server:$COUCHBASE_SERVER_VERSION}

export BATS_FORMATTER=${BATS_FORMATTER:-tap}
export BATS_ROOT=${BATS_ROOT:-$TEST_COMMON_SCRIPT_DIR/../tools/bats}
export BATS_FILE_ROOT=$BATS_ROOT/lib/bats-file
export BATS_SUPPORT_ROOT=$BATS_ROOT/lib/bats-support
export BATS_ASSERT_ROOT=$BATS_ROOT/lib/bats-assert
export BATS_DETIK_ROOT=$BATS_ROOT/lib/bats-detik

# shellcheck disable=SC1091
source "$HELPERS_ROOT/test-helpers.bash"

# Helper function to run a set of tests based on our specific configuration
# This function will call `exit`, so any cleanup must be done inside of it.
function run_tests() {
    local requested=$1
    local run=""
    local smoke=0

    if [[ "$requested" == "all" ]] || [ -z "$requested" ]; then
        # Empty => everything. Alternatively, explicitly ask for it.
        smoke=1
        run="--recursive ${TEST_ROOT}/smoke ${TEST_ROOT}/integration/${TEST_PLATFORM}"
    elif [[ "$requested" =~ .*\.bats$ ]]; then
        # One individual test
        run="$requested"
        if [[ "$requested" == *smoke* ]]; then
            smoke=1
        fi
    elif [[ "$requested" == "smoke" ]]; then
        # Smoke suite
        run="--recursive ${TEST_ROOT}/smoke"
        smoke=1
    elif [ -n "$requested" ]; then
        # Likely an individual integration suite
        run="--recursive ${TEST_ROOT}/$requested"
    fi

    if [ "$smoke" -eq 1 ]; then
        export SMOKE_NODES=3
        start_smoke_cluster
        trap teardown_smoke_cluster ERR EXIT
    fi

    echo
    echo
    echo "========================"
    echo "Starting tests."
    echo "========================"
    echo
    echo

    # We run BATS in a subshell to prevent it from inheriting our exit/err trap, which can mess up its internals
    # We set +exu because unbound variables can cause test failures with zero context
    set +xeu
    # shellcheck disable=SC2086
    (bats --formatter "${BATS_FORMATTER}" $run --timing)
    local bats_retval=$?

    echo
    echo
    echo "========================"
    if [ "$bats_retval" -eq 0 ]; then
        echo "All tests passed!"
    else
        echo "Some tests failed. Please inspect the output above for details."
    fi
    echo "========================"
    echo
    echo
    exit $bats_retval
}
