// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package config

import "go.uber.org/zap/zapcore"

type LogLevel string

func (l LogLevel) ToZap() zapcore.Level {
	result := zapcore.InfoLevel
	if err := result.UnmarshalText([]byte(l)); err != nil {
		panic(err)
	}
	return result
}
