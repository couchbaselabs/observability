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

# Waits for the given cURL call to succeed, waiting up to $1 times.
function wait_for_curl() {
    MAX_ATTEMPTS=$1
    shift
    ATTEMPTS=0
    until curl -s -o /dev/null -f "$@"; do
        if [ $ATTEMPTS -gt "$MAX_ATTEMPTS" ]; then
            fail "unable to perform cURL"
        fi
        ((ATTEMPTS++))
        sleep 2
    done
}

# Waits for the given URL to return 200
function wait_for_url() {
    MAX_ATTEMPTS=$1
    URL=$2
    CREDENTIALS=$3
    extra_args=""
    if [ -n "$CREDENTIALS" ]; then
        extra_args="-u $CREDENTIALS"
    fi
    # shellcheck disable=SC2086
    wait_for_curl "$MAX_ATTEMPTS" "$URL" $extra_args
}
