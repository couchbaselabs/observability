// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package utilities

import (
	"os"
	"path/filepath"
	"time"
)

// GetLastLinesOfEventLog gets the time of the last line of the events logs and returns the time along with a list of
// the lines that happened at that timestamp.
func GetLastLinesOfEventLog(nodeName string, prevPath string) (time.Time, []string, error) {
	file, err := os.Open(filepath.Join(prevPath, "events_"+nodeName+".log"))
	if err != nil {
		return time.Time{}, []string{}, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return time.Time{}, []string{}, err
	}

	size := stat.Size()
	if size <= 0 {
		return time.Time{}, []string{}, nil
	}

	var (
		line     string
		offset   = size - 1
		lines    []string
		lastTime time.Time
	)

	// get the last line of the file and keep going backwards until it gets all of the lines with the same timestamp as
	// the very last line of the file and add these lines to a list.
	for {
		// return if at start of file
		if offset <= 0 {
			return lastTime, lines, nil
		}

		line, offset, err = getLine(offset-1, file)
		if err != nil {
			return time.Time{}, []string{}, err
		}

		timestamp, err := GetTime(line)
		if err != nil {
			return time.Time{}, []string{}, err
		}

		// break if not at the same time as the time of the very last line
		if !timestamp.Equal(lastTime) && !lastTime.IsZero() {
			break
		}

		lines = append(lines, line)
		lastTime = timestamp
	}

	return lastTime, lines, nil
}

// getLine searches backwards starting from the index given by offset to get the last line of the file by reading
// each character looking for a newline rune.
func getLine(offset int64, file *os.File) (string, int64, error) {
	var start int64
	var line string
	char := make([]byte, 1)

	for start = offset; start >= 0; start-- {
		_, err := file.ReadAt(char, start)
		if err != nil {
			return "", 0, err
		}

		if char[0] == '\n' || char[0] == '\r' {
			break
		}

		line = string(char[0]) + line
	}

	return line, start, nil
}
