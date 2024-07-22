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

# Start Loki, run run_single_test.sh for each test case, then clean up.
# Logs will be written to a Loki tenant named the same as the test suite, to ensure tests are isolated from each other.
#
# Usage: run_all.sh [OPTIONS]
#
# Options:
#     -v, -vv,-vvv, --verbosity <value>:    Increase verbosity. The higher the value, the more verbose the output will be.
#     -s, --skip-teardown:                  Don't remove Loki container after exit. (Env var: SKIP_TEARDOWN)
#     -l <url>, --loki <url>:               Use an already running Loki instead of starting one up. (Env var: LOKI_HOST)
#     --loki-port <port>:                   Use a non-default Loki port.

set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
HELPERS_ROOT="$SCRIPT_DIR/../helpers"
# shellcheck disable=SC1091
source "$HELPERS_ROOT/url-helpers.bash"

VERBOSITY=0
SKIP_TEARDOWN=${SKIP_TEARDOWN:-false}
LOKI_PORT=3100

while [[ $# -gt 0 ]]; do
  case $1 in
    --verbosity)
        VERBOSITY="$2"
        shift
        shift
        ;;
    -vv)
        VERBOSITY=2
        shift
        ;;
    -v)
        VERBOSITY=1
        shift
        ;;
    -s|--skip-teardown)
        SKIP_TEARDOWN="true"
        shift
        ;;
    -l|--loki)
        LOKI_HOST="$2"
        shift
        shift
        ;;
    --loki-port)
        LOKI_PORT="$2"
        shift
        shift
        ;;
  esac
done

export VERBOSITY=$VERBOSITY

# Helper to only print at the given verbosity or above.
# Parameters:
# $1: min verbosity to print
# all others: echoed
function log() {
    if [ "$VERBOSITY" -lt "$1" ]; then
        return
    fi
    shift
    echo "$@"
}

if [ -z "${LOKI_HOST:-}" ]; then
    # Start up Loki
    log 1 "Starting Loki..."
    loki_container_id=$(docker run --rm -d -p 3100:3100 --name test_loki grafana/loki:2.4.1 -config.file=/etc/loki/local-config.yaml -log.level=debug)
    log 1 "Waiting for Loki to become ready..."
    wait_for_url 10 http://localhost:3100/ready
    log 2 "Loki ready."
    LOKI_HOST="localhost"
    LOKI_PORT=3100
fi

exit_code=0

for case in "$SCRIPT_DIR"/*/*; do
    log 1 "RUN ${case##"$SCRIPT_DIR/"}"
    set +e
    VERBOSITY="$VERBOSITY" LOKI_TEST_RUN_ALL=true "$SCRIPT_DIR/run_single_test.sh" --skip-teardown -l "$LOKI_HOST" --loki-port "$LOKI_PORT" "${case##"$SCRIPT_DIR/"}"
    test_exit=$?
    if [ "$test_exit" -ne 0 ]; then
        exit_code="$test_exit"
    fi
done

if [ "$SKIP_TEARDOWN" != "true" ]; then
    if docker inspect test_loki >/dev/null 2>&1; then
        docker rm -f "$loki_container_id" >/dev/null
    fi
else
    log 1 "Skipping Loki teardown. To manually tear it down, run docker rm -f test_loki && docker network rm loki_alerts_test"
fi

exit "$exit_code"
