// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import "fmt"

type AuthError struct {
	Authentication bool
	err            error
}

func (e AuthError) Error() string {
	return fmt.Sprintf("invalid auth: %v", e.err)
}

func (e AuthError) Unwrap() error {
	return e.err
}
