// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import "errors"

var (
	ErrNotInLine           = errors.New("event not in this line")
	ErrNotFullLine         = errors.New("part of line is missing; getting next section of line")
	ErrRegexpMissingFields = errors.New("not all regexp capture groups found")
	ErrAlreadyInLog        = errors.New("line already in events log")
)
