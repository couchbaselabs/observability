// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import "errors"

// ErrNotFound is a generic error for when we cannot get a resource.
var ErrNotFound = errors.New("resource not found")
