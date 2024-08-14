// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import "time"

// Credential represents a set of credentials for the Couchbase Cloud API.
type Credential struct {
	Name      string    `json:"name"`
	AccessKey string    `json:"access_key,omitempty"`
	SecretKey string    `json:"secret_key,omitempty"`
	DateAdded time.Time `json:"date_added"`
}
