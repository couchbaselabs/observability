// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import "time"

// FrequencyConfiguration is just a convenient grouping of all the frequencies for the different monitors the manger
// runs.
type FrequencyConfiguration struct {
	Heart                time.Duration
	Status               time.Duration
	Janitor              time.Duration
	DiscoveryRunsTimeGap time.Duration
	AgentPortReconcile   time.Duration
}
