// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	_ "embed"
	"encoding/json"
)

//go:embed rules.json
var embeddedRules []byte

func loadEmbeddedRules() ([]rule, error) {
	var result []rule
	err := json.Unmarshal(embeddedRules, &result)
	return result, err
}
