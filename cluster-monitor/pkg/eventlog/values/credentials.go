// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

type Credentials struct {
	User     string
	Password string
	Cluster  string
	NodeName string
}
