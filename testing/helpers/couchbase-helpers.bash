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

# Returns the exposed port of a Docker container (usually used with a Docker Compose service).
# Arguments:
# $1: the name of the container, or part of the name
# $2: the container port to find the host counterpart of
function get_service_port() {
    ports=$(docker ps --filter "name=$1" --format "{{.Ports}}")
    echo "${ports}" | sed -e 's/, /\n/g' | perl -ne 'print "$1" if /0.0.0.0:(\d+)->'"$2"'/'
}
