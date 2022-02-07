#!/usr/bin/env bats

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

load "$HELPERS_ROOT/test-helpers.bash"

ensure_variables_set TEST_ROOT CMOS_PORT BATS_SUPPORT_ROOT BATS_ASSERT_ROOT BATS_FILE_ROOT HELPERS_ROOT COUCHBASE_SERVER_IMAGE

load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"
load "$HELPERS_ROOT/couchbase-helpers.bash"
load "$HELPERS_ROOT/url-helpers.bash"

setup_file() {
    timeout 180 docker-compose --project-directory="${BATS_TEST_DIRNAME}" up -d --force-recreate --remove-orphans
}

teardown_file() {
    run docker-compose --project-directory="${BATS_TEST_DIRNAME}" logs --timestamps || echo "Unable to get compose output"
    if [ "${SKIP_TEARDOWN:-false}" == "true" ]; then
        skip "Skipping teardown"
    elif [ "${TEST_NATIVE:-false}" == "true" ]; then
        run docker-compose --project-directory="${BATS_TEST_DIRNAME}" rm -v --force --stop
    fi
}

@test "Verify that Prometheus uses the path prefix" {
    wait_for_url 12 "http://localhost:${CMOS_PORT}/cmos/prometheus/-/ready"
}

@test "Verify that Alertmanager uses the path prefix" {
    wait_for_url 12 "http://localhost:${CMOS_PORT}/cmos/alertmanager/-/healthy"
}

@test "Verify that Loki uses the path prefix" {
    wait_for_url 12 "http://localhost:${CMOS_PORT}/cmos/loki/api/v1/status/buildinfo"
}

@test "Verify that Grafana uses the path prefix" {
    wait_for_url 12 "http://localhost:${CMOS_PORT}/cmos/grafana/api/health"
}

@test "Verify that the docs are accessible on the path prefix" {
    run curl -fsS "http://localhost:${CMOS_PORT}/cmos/docs/cmos/0.1/index.html"
}
