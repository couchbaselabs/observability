// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package scraper

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/parsers"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/utilities"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"

	"go.uber.org/zap"
)

type logFile struct {
	Name   string
	Line   string
	Reader *bufio.Reader
}

// MergeEventLogs merges the event logs for each of the CB logs to create a final time orderd event log.
func MergeEventLogs(cred *values.Credentials, outputPath string) error {
	linePointers := []*logFile{}

	eventLog, err := os.OpenFile(filepath.Join(outputPath, "events_"+cred.NodeName+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer eventLog.Close()

	for _, log := range parsers.ParserFunctions {
		file, err := os.Open(filepath.Join(outputPath, log.Name+"_events.log"))
		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed to open event log", "log", log.Name, "err", err)
			continue
		}
		defer file.Close()

		logfile, err := newLogFile(log.Name, file)
		if err == io.EOF {
			continue
		}

		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed get first line from event log", "log", log.Name, "err", err)
			continue
		}

		linePointers = append(linePointers, logfile)
	}

	for len(linePointers) > 0 {
		name, err := mergeNextLine(eventLog, linePointers)
		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed to merge line into events log", "err", err)
		}

		index := -1

		for i, file := range linePointers {
			if name == file.Name {
				nextLine, _, err := file.Reader.ReadLine()
				if err == io.EOF {
					index = i
					break
				} else if err != nil {
					zap.S().Warnw("(SCRAPER) Failed to merge line into events log", "log", file.Name, "err", err)
					break
				}

				file.Line = string(nextLine)
			}
		}

		if index != -1 {
			linePointers = append(linePointers[:index], linePointers[index+1:]...)
		}
	}

	return nil
}

// mergeNextLine finds the next line and writes it to the event log and returns name of the file the line was taken
// from.
func mergeNextLine(eventLog *os.File, nextLines []*logFile) (string, error) {
	var (
		earliestTime time.Time
		earliestLine string
		earliestName string
	)

	// get earliest time from the next lines of each of the logs
	for _, next := range nextLines {
		timestamp, err := utilities.GetTime(next.Line)
		if err != nil {
			return next.Name, err
		}

		if timestamp.Before(earliestTime) || earliestTime.IsZero() {
			earliestTime = timestamp
			earliestName = next.Name
			earliestLine = next.Line
		}
	}

	// write line with earliest time to event log
	_, err := eventLog.WriteString(earliestLine + "\n")
	if err != nil {
		return earliestName, err
	}

	return earliestName, nil
}

// newLogFile creates a logFile struct for a given log file.
func newLogFile(logName string, file *os.File) (*logFile, error) {
	var line []byte
	var err error

	reader := bufio.NewReader(file)

	line, _, err = reader.ReadLine()
	if err == io.EOF {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &logFile{
		Name:   logName,
		Line:   string(line),
		Reader: reader,
	}, nil
}
