#!/usr/bin/env bats
# shellcheck disable=SC2034

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

# The intention of this file is to verify the tooling installed within the container.
# This is so that it can then be used by actual tests.

load "$BATS_DETIK_ROOT/utils.bash"
load "$BATS_DETIK_ROOT/linter.bash"
load "$BATS_DETIK_ROOT/detik.bash"
load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

@test 'check bats_assert' {
    assert true
    touch '/tmp/test.log'
    assert [ -e '/tmp/test.log' ]
    rm -f '/tmp/test.log'
    refute [ -e '/tmp/test.log' ]
}

@test 'check bats_file' {
    assert_dir_exist /tmp
    assert_file_not_exist /I/do/not/exist.please
}

@test "check kubectl" {
    # This can run regardless of whether we are in k8s or not.
	kubectl version --client=true
}

@test "check helm" {
    helm version
}

@test 'check bats_detik' {
    if [[ "$TEST_NATIVE" == "true" ]]; then
        skip "Skipping kubernetes specific tests"
    fi
    # For whatever reason namespace must be provided with the client name
    DETIK_CLIENT_NAME="kubectl -n kube-system"
    DETIK_CLIENT_NAMESPACE="kube-system"
    verify "there is 1 service named 'kube-dns'"
}

@test 'check docker-compose' {
    docker-compose version
}