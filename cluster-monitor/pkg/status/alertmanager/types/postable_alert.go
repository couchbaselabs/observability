// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package types

import "time"

// PostableAlert is the alert data structure for the Alertmanager API.
// See: https://github.com/prometheus/compliance/blob/main/alert_generator/specification.md#alert-format
type PostableAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	StartsAt     *time.Time        `json:"startsAt,omitempty"`
	EndsAt       *time.Time        `json:"endsAt,omitempty"`
}
