// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package tunables

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// TODO (CMOS-378): once we're on Go 1.18 this can use generics

func integer(name string, def int) int {
	valueStr := os.Getenv(varsPrefix + name)
	if valueStr == "" {
		return def
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		panic(fmt.Errorf("invalid value for %s: %w", varsPrefix+name, err))
	}
	zap.S().Infow("Using non-default tunable value", "name", varsPrefix+name, "value", value)
	return value
}

func duration(name string, def time.Duration) time.Duration {
	valueStr := os.Getenv(varsPrefix + name)
	if valueStr == "" {
		return def
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		panic(fmt.Errorf("invalid value for %s: %w", varsPrefix+name, err))
	}
	zap.S().Infow("Using non-default tunable value", "name", varsPrefix+name, "value", value)
	return value
}
