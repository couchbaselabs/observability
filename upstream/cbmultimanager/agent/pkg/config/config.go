// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package config

import (
	"time"
)

type Config struct {
	HTTPPort             int
	CheckInterval        time.Duration
	CouchbaseInstallPath string
	LogLevel             LogLevel
	CouchbaseUsername    string
	CouchbasePassword    string
	AutoFeatures         bool
	EnableFeatures       []string
	DisableFeatures      []string
	JanitorInterval      time.Duration
	LogAlertDuration     time.Duration
}
