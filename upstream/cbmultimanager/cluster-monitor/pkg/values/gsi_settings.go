// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"encoding/json"
	"fmt"
	"strings"
)

type GSIStorageMode string

const (
	MemoryOptimized GSIStorageMode = "memory_optimized"
	ForestDB        GSIStorageMode = "forestdb"
	Plasma          GSIStorageMode = "plasma"
)

type GSILogLevel int16

const (
	Silent GSILogLevel = iota
	Fatal
	Error
	Warn
	Info
	Verbose
	Timing
	Debug
	Trace
)

const DefaultGSILogLevel = Info

func (g GSILogLevel) String() string {
	switch g {
	case Silent:
		return "Silent"
	case Fatal:
		return "Fatal"
	case Error:
		return "Error"
	case Warn:
		return "Warn"
	case Info:
		return "Info"
	case Verbose:
		return "Verbose"
	case Timing:
		return "Timing"
	case Debug:
		return "Debug"
	case Trace:
		return "Trace"
	default:
		return "Info"
	}
}

func (g GSILogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.String())
}

func (g *GSILogLevel) UnmarshalJSON(data []byte) error {
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return err
	}
	switch strings.ToUpper(strVal) {
	case "SILENT":
		*g = Silent
	case "FATAL":
		*g = Fatal
	case "ERROR":
		*g = Error
	case "WARN":
		*g = Warn
	case "INFO":
		*g = Info
	case "VERBOSE":
		*g = Verbose
	case "TIMING":
		*g = Timing
	case "DEBUG":
		*g = Debug
	case "TRACE":
		*g = Trace
	default:
		return fmt.Errorf("unknown GSI log level %s", strVal)
	}
	return nil
}

// GSISettings represents the output of the /indexer/settings endpoint.
type GSISettings struct {
	RedistributeIndexes    bool           `json:"redistributeIndexes"`
	NumReplicas            int            `json:"numReplica"`
	IndexerThreads         int            `json:"indexerThreads"`
	MemorySnapshotInterval int            `json:"memorySnapshotInterval"`
	StableSnapshotInterval int            `json:"stableSnapshotInterval"`
	MaxRollbackPoints      int            `json:"maxRollbackPoints"`
	LogLevel               GSILogLevel    `json:"logLevel"`
	StorageMode            GSIStorageMode `json:"storageMode"`
}
