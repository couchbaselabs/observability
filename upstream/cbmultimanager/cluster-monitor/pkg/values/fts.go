// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

type FTSIndexStatus struct {
	Status    string       `json:"status"`
	IndexDefs FTSIndexDefs `json:"indexDefs"`
}

type FTSIndexDefs struct {
	IndexDefs map[string]SingleFTSIndex `json:"indexDefs"`
}

type SingleFTSIndex struct {
	Name           string        `json:"name"`
	SourceName     string        `json:"sourceName"`
	PlanParameters FTSPlanParams `json:"planParams"`
}

type FTSPlanParams struct {
	NumReplicas int `json:"numReplicas,omitempty"`
}
