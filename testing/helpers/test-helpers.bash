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

# Verifies if all the given variables are set, and exits otherwise
# Parameters:
# Variadic: variable names to check presence of
function ensure_variables_set() {
    missing=""
    for var in "$@"; do
        if [ -z "${!var}" ]; then
            missing+="$var "
        fi
    done
    if [ -n "$missing" ]; then
        # We use exit rather than fail so that this works even if the BATS helper root is missing
        echo "Missing required variables: $missing"
        exit 1
    fi
}

# Finds a random, unused port on the system and assigns it to the given variable. Exits immediately if it can't find one.
# Parameters:
# $1: The name of a variable to assign the port to.
function find_unused_port() {
    local varname="$1"
    local portnum
    while true; do
        portnum=$(shuf -i 1025-65535 -n 1)
        if ! lsof -Pi ":$portnum" -sTCP:LISTEN; then
            declare "$varname=$portnum"
            return 0
        fi
    done
}
