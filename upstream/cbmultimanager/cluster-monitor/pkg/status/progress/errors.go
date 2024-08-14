// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package progress

import "fmt"

type ClusterNotFoundError struct {
	uuid string
}

func (e *ClusterNotFoundError) Error() string {
	return fmt.Sprintf("cluster '%s' not found", e.uuid)
}
