package scraper

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/log/parsers"
	"github.com/couchbaselabs/cbmultimanager/log/utilities"
	"github.com/couchbaselabs/cbmultimanager/log/values"

	"go.uber.org/zap"
)

var commaEndRegexp = regexp.MustCompile(`.*(:|-\>|,)$`)

// RunParsers runs all the parsers for a given CB log and outputs the results to events_[node_name].log file.
func RunParsers(file parsers.Log, cred *values.Credentials, cbcollectPath string,
	logPath string) error {
	eventLog, err := os.Create(file.Name + "_events.log")
	if err != nil {
		return err
	}
	defer eventLog.Close()

	if err = getLogFiles(file.Name, cred, cbcollectPath, logPath); err != nil {
		return err
	}

	log, err := os.Open(filepath.Join(logPath, file.Name+".log"))
	if err != nil {
		return err
	}
	defer log.Close()

	reader := bufio.NewReader(log)
	if err != nil {
		return err
	}

	continueTime, continueLines, err := utilities.GetLastLinesOfEventLog(cred.NodeName)
	if err != nil {
		zap.S().Warnw("(SCRAPER) Failed to get last line of previous event log; will create new full event log", "error",
			err)
		continueTime = time.Time{}
	}

	var (
		fullLine      string
		diagEnd       int
		previousError bool
	)

	for {
		fullLine, diagEnd, err = runParsersOnLine(reader, fullLine, continueTime, continueLines, eventLog, file, diagEnd)
		if errors.Is(err, io.EOF) {
			break
		}

		if errors.Is(err, values.ErrAlreadyInLog) || errors.Is(err, values.ErrNotInLine) {
			continue
		}

		if err != nil {
			if previousError {
				return fmt.Errorf("error while reading log %s", file.Name)
			}

			previousError = true
			continue
		}

		previousError = false
	}

	return nil
}

// getLogFiles gets the Couchbase logs either by using REST calls or by unzipping a CBcollect.
func getLogFiles(filename string, cred *values.Credentials, cbcollectPath string, logPath string) error {
	if cbcollectPath == "" {
		return utilities.GetLog(filename, filepath.Join(logPath, filename+".log"), cred)
	}

	return utilities.Unzip(cbcollectPath, logPath)
}

// runParsersOnLine runs all parsers and writes any events found to the events log for each line of the CB log.
// It takes the CB log reader, the line so far, the line and time of the last line of the previous events log, an open
// event log and an int that when equal to two shows it's reached the end of the neccesery part of the diag log.
func runParsersOnLine(reader *bufio.Reader, fullLine string, continueTime time.Time, continueLines []string,
	eventLog *os.File, file parsers.Log, diagEnd int) (string, int, error) {
	var notFullLine bool
	line, isPrefix, err := reader.ReadLine()
	if errors.Is(err, io.EOF) {
		return "", diagEnd, err
	}

	if err != nil {
		zap.S().Warnw("(SCRAPER) Could not read line of log file", "file", file.Name, "error", err)
		return "", diagEnd, err
	}

	fullLine += strings.TrimSpace(string(line))

	if file.Name == "diag" && strings.HasPrefix(fullLine, "---------------") {
		diagEnd++
		if diagEnd > 1 {
			return "", diagEnd, io.EOF
		}
	}

	if len(commaEndRegexp.FindAllString(string(line), -1)) > 0 || isPrefix || fullLine == "" {
		return fullLine, diagEnd, nil
	}

	timestamp, err := utilities.GetTimeFromString(file.StartsWithTime, fullLine, file.Name)
	if err != nil {
		return "", diagEnd, values.ErrNotInLine
	}

	// ignore any lines that are already in events log
	if timestamp.Before(continueTime) {
		return "", diagEnd, values.ErrAlreadyInLog
	}

	for _, function := range file.Parsers {
		event, err := function(fullLine)
		if errors.Is(err, values.ErrNotInLine) {
			continue
		}

		if errors.Is(err, values.ErrNotFullLine) {
			notFullLine = true
			break
		}

		if err != nil {
			zap.S().Warnw("(SCRAPER) Parser failed to run", "error", err)
			break
		}

		event.Time = timestamp

		if err = writeLine(event, timestamp, continueTime, continueLines, eventLog); err != nil {
			zap.S().Warnw("(SCRAPER) Failed to write line to log", "error", err)
			break
		}
	}

	if !notFullLine {
		fullLine = ""
	}

	return fullLine, diagEnd, nil
}

// writeLine writes an event line for an event found by a parser to the event log.
func writeLine(event *values.Result, timestamp time.Time, continueTime time.Time, continueLines []string,
	eventLog *os.File) error {
	jsonResult, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if timestamp.Equal(continueTime) {
		var repeatedLine bool
		for _, line := range continueLines {
			if string(jsonResult) == line {
				repeatedLine = true
			}
		}

		if repeatedLine {
			return values.ErrAlreadyInLog
		}
	}

	_, err = eventLog.WriteString(string(jsonResult) + "\n")
	if err != nil {
		return err
	}

	return nil
}
