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
set -u

echo "Waiting for server startup"
ATTEMPTS=0
# Wait for startup - no great way for this
until curl -u "Administrator:password" http://db1:8091/pools/default &> /dev/null; do
    # Prevent an infinite loop - at 2 seconds per go this is 5 minutes
    if [ $ATTEMPTS -gt "150" ]; then
        echo "Unable to start up Couchbase Server"
        exit 1
    fi
    ATTEMPTS=$((ATTEMPTS+1))
    echo "Not running, waiting to recheck"
    sleep 2
done

echo "Running workload generation"
while true; do
    cbworkloadgen -n db1:8091 -b testBucket -u Administrator -p password
    sleep 10
done
echo "Exiting"
