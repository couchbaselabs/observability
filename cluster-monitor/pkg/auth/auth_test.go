// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	password := "my-secret-password"
	out, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Unexpected error hashing password: %v", err)
	}

	// stupid check to ensure we are not given the password un hashed or padded
	if strings.Contains(string(out), password) {
		t.Fatalf("'%s' is not hashed properly", string(out))
	}

	if !CheckPassword(password, out) {
		t.Fatal("Expected password to match")
	}

	out[0]++
	if CheckPassword(password, out) {
		t.Fatal("Expected password not to match")
	}
}
