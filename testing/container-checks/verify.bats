#!/usr/bin/env bats

load "$BATS_DETIK_ROOT/utils.bash"
load "$BATS_DETIK_ROOT/linter.bash"
load "$BATS_DETIK_ROOT/detik.bash"
load "$BATS_SUPPORT_ROOT/load.bash"
load "$BATS_ASSERT_ROOT/load.bash"
load "$BATS_FILE_ROOT/load.bash"

DETIK_CLIENT_NAME="kubectl"

@test "check kubectl" {
	kubectl version --client=true
}

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

@test 'check bats_detik' {
    DETIK_CLIENT_NAME="kubectl -n kube-system"
    DETIK_CLIENT_NAMESPACE="kube-system"
    verify "there are 1 services named 'kube-dns'"
}