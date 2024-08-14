// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package logger

import (
	"fmt"

	"github.com/couchbase/tools-common/log"
	"go.uber.org/zap"
)

type ToolsCommonLogger struct {
	logger *zap.SugaredLogger
}

func NewToolsCommonLogger(l *zap.SugaredLogger) *ToolsCommonLogger {
	return &ToolsCommonLogger{logger: l}
}

func (t ToolsCommonLogger) Log(level log.Level, format string, args ...any) {
	tag := zap.Bool("tools-common", true)
	switch level {
	case log.LevelTrace:
		// No 'Trace' equivalent for Logger interface in couchbase cloud ∴ default to debug
		t.logger.Debug(fmt.Sprintf(format, args...), tag)
	case log.LevelDebug:
		t.logger.Debug(fmt.Sprintf(format, args...), tag)
	case log.LevelInfo:
		t.logger.Info(fmt.Sprintf(format, args...), tag)
	case log.LevelWarning:
		t.logger.Warn(fmt.Sprintf(format, args...), tag)
	case log.LevelError:
		t.logger.Error(fmt.Sprintf(format, args...), tag)
	case log.LevelPanic:
		// NOTE: This functions panics, this means our panic takes president over the tools-common panic
		t.logger.Fatal(fmt.Sprintf(format, args...), tag)
	}
}
