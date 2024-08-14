// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

// User is a basic representation of a multi cluster user.
type User struct {
	User     string `json:"user"`
	Password []byte `json:"-"`
	Admin    bool   `json:"admin"`
}
