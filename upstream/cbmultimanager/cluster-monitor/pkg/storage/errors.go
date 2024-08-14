// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package storage

import "errors"

// ErrUserAlreadyExists is returned by (*storage.Store).AddUser when a user with that username already exists.
var ErrUserAlreadyExists = errors.New("user already exists")
