// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// iso8601TimeEncoder serializes a time.Time to an ISO8601-formatted string.
func iso8601TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02T15:04:05.000Z07:00"))
}

func Init(level zapcore.Level, logDir string) error {
	var logFile *os.File
	if logDir != "" {
		var err error
		logFile, err = openLogFile(logDir)
		if err != nil {
			return err
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = iso8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeName = func(name string, enc zapcore.PrimitiveArrayEncoder) {
		if len(name) == 0 {
			return
		}
		enc.AppendString("(" + name + ")")
	}
	encoderConfig.ConsoleSeparator = " "

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	var writer zapcore.WriteSyncer
	if logFile != nil {
		writer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(logFile), zapcore.AddSync(os.Stdout))
	} else {
		writer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writer, level)
	zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)

	return nil
}

func openLogFile(logDir string) (*os.File, error) {
	err := os.MkdirAll(logDir, 0o770)
	if err != nil {
		return nil, fmt.Errorf("could not create log directory: %w", err)
	}

	logFile, err := os.OpenFile(filepath.Join(logDir, "cbmultimanger.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}

	return logFile, nil
}
