package utilities

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// GetTime gets the timestamp from a JSON line.
func GetTime(line string) (time.Time, error) {
	type info struct {
		Time time.Time `json:"timestamp"`
	}

	var timeInfo info

	if err := json.Unmarshal([]byte(line), &timeInfo); err != nil {
		return time.Time{}, err
	}

	return timeInfo.Time, nil
}

// GetTimeString gets the time from a line from any CB log.
func GetTimeFromString(start bool, line string, logName string) (time.Time, error) {
	var startTime string
	if !start && line[0] == '[' {
		// get time from line eg: [menelaus:info,2021-02-19T13:29:37.95Z,ns_1@10.144.210.101:<0.27363.671>:
		//	menelaus_web_buckets:handle_bucket_delete:408]Deleted bucket "beer-sample"
		timeSlice := strings.Split(line, ",")
		if len(timeSlice) < 2 {
			return time.Time{}, values.ErrNotInLine
		}

		startTime = timeSlice[1]
	} else if start && line[0] >= '0' && line[0] <= '9' {
		// get time from line eg: 2021-02-19T13:29:37.95Z [Info] Indexer::NewIndexer Status Active
		timeSlice := strings.Fields(line)
		if len(timeSlice) < 2 {
			return time.Time{}, values.ErrNotInLine
		}

		startTime = timeSlice[0]
	} else {
		// return error for any other line eg: cbbrowse_logs info.log
		return time.Time{}, values.ErrNotInLine
	}

	if logName == "diag" {
		startTime = startTime[:len(startTime)-1]
	}

	timestamp, err := time.Parse(time.RFC3339Nano, startTime)
	if err != nil {
		return time.Time{}, err
	}

	return timestamp, nil
}
