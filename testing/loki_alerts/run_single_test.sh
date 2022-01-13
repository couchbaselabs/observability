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

# Run a single Loki alerts test suite.
# Uses Fluent Bit to read in the test data, then runs the alert's query.
# Logs will be written to a Loki tenant named the same as the test suite, to ensure tests are isolated from each other.
#
# Usage: run_single_test.sh [OPTIONS] Group-Name/testName
#
# Options:
#     -v, -vv,-vvv, --verbosity <value>:    Increase verbosity. The higher the value, the more verbose the output will be.
#     -s, --skip-teardown:                  Don't remove Loki container after exit. (Env var: SKIP_TEARDOWN)
#     -l <host>, --loki <host>:             Use an already running Loki instead of starting one up. (Env var: LOKI_HOST)
#     --loki-port <port>:                   Use a non-default Loki port.

set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
HELPERS_ROOT="$SCRIPT_DIR/../helpers"
# shellcheck disable=SC1091
source "$HELPERS_ROOT/url-helpers.bash"

VERBOSITY=${VERBOSITY:-0}
SKIP_TEARDOWN=${SKIP_TEARDOWN:-false}
LOKI_PORT=${LOKI_PORT:-3100}

while [[ $# -gt 1 ]]; do
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

if [ "$#" -lt 1 ]; then
    echo "Usage: $(basename "$0") Group-Name/alertName"
    exit 1
fi

# Helper that maps a Couchbase log file to the Fluent Bit config include used for it
# Parameters:
# $1: the Couchbase log file we're working with
# Echoes the basename (no path) of the file you'd include to parse it
function fluent_bit_config_for() {
    case $1 in
        memcached.log.000000.txt)
            echo "in-memcached-log.conf"
            ;;
        babysitter.log)
            echo "in-erlang-multiline-log.conf"
            ;;
        *)
            >&2 echo "ERROR: unknown fluent bit config for $1"
            exit 1
            ;;
    esac
}

suite="$1"
group=$(echo "$suite" | sed -e 's/\/.*//')
alert=$(echo "$suite" | sed -e 's/.*\///')

log 2 "Running suite: $suite"

logs_path="$SCRIPT_DIR/$group/$alert"
if [ ! -d "$logs_path" ]; then
    echo "$group/$alert has no log files."
    exit 1
fi

expr=$(yq e -e ".groups[] | select(.name == "'"'"$group"'"'") | .rules[] | select(.alert == "'"'"$alert"'"'") | .expr" "$SCRIPT_DIR/../../microlith/loki/alerting/couchbase/couchbase-rules.yaml")

if [ -z "${LOKI_HOST:-}" ]; then
    # Start up Loki
    log 2 "Starting Loki..."
    loki_container_id=$(docker run --rm -d -p 3100:3100 --name test_loki grafana/loki:2.4.1 -config.file=/etc/loki/local-config.yaml -log.level=debug)
    LOKI_HOST="localhost"
    LOKI_PORT=3100
fi

log 1 "Waiting for Loki to become ready..."
wait_for_url 10 http://localhost:3100/ready
log 2 "Loki ready."

# Generate the Fluent Bit config - we can't just reuse it, because we need to set Read_from_Head or it'll skip straight to the end of the files.
# Three main parts to it: a header and a trailer, and in between them a section for each config file we're reading]
log 2 "Generating fluent-bit config..."
fb_cfg=$(cat "$SCRIPT_DIR/fluent-bit.header.conf")

for f in "$logs_path"/*; do
    fb_cfg="$fb_cfg

@include /fluent-bit/etc/couchbase/$(fluent_bit_config_for "$(basename "$f")")
    Read_from_Head True
"
done

fb_cfg="$fb_cfg
$(cat "$SCRIPT_DIR/fluent-bit.trailer.conf")
"

config_dir=$(mktemp -d)
echo "$fb_cfg" > "$config_dir/fluent-bit.conf"
log 2 "Fluent Bit config generated and written to $config_dir/fluent-bit.conf"

# Run Fluent Bit over the case's files
log 1 "Starting Fluent Bit..."
fb_command="docker run --rm -d \
    --network host \
    -v "'"'"$logs_path:/opt/couchbase/var/lib/couchbase/logs:ro"'"'" \
    -v "'"'"$config_dir/fluent-bit.conf:/fluent-bit/config/fluent-bit.conf"'"'" \
    -v "'"'"$SCRIPT_DIR/rebase_times.lua:/fluent-bit/config/rebase_times.lua"'"'" \
    -e 'LOKI_MATCH=*' \
    -e 'LOKI_HOST=$LOKI_HOST' \
    -e 'LOKI_PORT=$LOKI_PORT' \
    -e 'LOKI_TENANT=$suite' \
    couchbase/fluent-bit:1.1.3"
log 2 "Command: $fb_command"
fb_container_id=$(eval "$fb_command")

# Print the logs so we see what's going on.
# Would like to use Exit_on_EOF, but https://github.com/fluent/fluent-bit/issues/3274
# Instead we just run FB and wait a bit - long enough for it to dump the full logs under any reasonable circumstances
fb_execution_timeout_seconds=10
if [ "$VERBOSITY" -lt 2 ]; then
    sleep "$fb_execution_timeout_seconds"
else
    # timeout will abort the script with error 124 without the ||true
    timeout "$fb_execution_timeout_seconds" docker logs -ft "$fb_container_id" || true
fi
docker rm -f "$fb_container_id"

# Now, execute the query against Loki and see if we get anything out
tmpdir=$(mktemp -d)
# 2 hours ago, as nanoseconds since the Unix epoch
query_range_start=$(( $(date +%s) - 2 * 60 * 60))000000000
# dto, but 1 hour in the future, just in case
query_range_end=$(( $(date +%s) + 60 * 60))000000000
if [ "$VERBOSITY" -lt 3 ]; then
    curl_flags="-fsS"
else
    curl_flags="-fv"
fi
curl "$curl_flags" -o "$tmpdir/query-result.json" -G \
    -H "X-Scope-OrgID: $suite" \
    --data-urlencode "query=$expr" \
    --data-urlencode "start=$query_range_start" \
    --data-urlencode "end=$query_range_end" \
    --data-urlencode "direction=BACKWARD" \
    --data-urlencode "step=1" \
    "http://$LOKI_HOST:$LOKI_PORT/loki/api/v1/query_range"

# Don't need Loki anymore, kill it now
if [ "$SKIP_TEARDOWN" != "true" ]; then
    if docker inspect test_loki >/dev/null 2>&1; then
        docker rm -f "$loki_container_id" >/dev/null
    fi
elif [ -z "${LOKI_TEST_RUN_ALL:-}" ]; then
    log 1 "Skipping Loki teardown. To manually tear it down, run docker rm -f test_loki && docker network rm loki_alerts_test"
fi

results=$(jq -r '.data.result | length' "$tmpdir/query-result.json")
rm -rf "$tmpdir"

log 2 "JQ result: $results"

if [ -z "$results" ] || [ "$results" -eq 0 ]; then
    echo "FAIL $suite"
    exit 1
else
    echo "PASS $suite"
    exit 0
fi
