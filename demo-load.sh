#!/usr/bin/env bash
set -exuo pipefail

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

container_id=$(kubectl get pods -l couchbase_cluster=cb7 -o go-template="{{with index .items 0 }}{{with index .status.containerStatuses 0}}{{.containerID}}{{end}}{{end}}" | sed -e 's/docker:\/\///')

docker cp demo-queries.txt "$container_id":/demo-queries.txt

{ docker exec -i "$container_id" /opt/couchbase/bin/cbc-pillowfight -U couchbase://localhost/travel-sample -u Administrator -P password -r 20 -R -M 1024 -J & } 1>/dev/null 2>&1

{ docker exec -i "$container_id" /opt/couchbase/bin/cbc-n1qlback -U couchbase://localhost/travel-sample -u Administrator -P password -t 2 -f demo-queries.txt & } 1>/dev/null 2>&1
