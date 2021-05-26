package scraper

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/couchbaselabs/cbmultimanager/log/values"

	"go.uber.org/zap"
)

type eventStruct struct {
	Event values.EventType `json:"event_type"`
}

// FilterEvents creates an events file containing only including or excluding the given events.
func FilterEvents(cred *values.Credentials, events []values.EventType, include bool) error {
	eventLog, err := os.Open("events_" + cred.NodeName + ".log")
	if err != nil {
		return err
	}
	defer eventLog.Close()

	filteredEventLog, err := os.Create("filtered_events_" + cred.NodeName + ".log")
	if err != nil {
		return err
	}
	defer filteredEventLog.Close()

	reader := bufio.NewReader(eventLog)

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed to read line from events.log", "err", err)
			continue
		}

		var eventInfo eventStruct

		err = json.Unmarshal(line, &eventInfo)
		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed to unmarshal line from events.log", "err", err)
			continue
		}

		wrongEvent := include

		for _, event := range events {
			if event == eventInfo.Event {
				wrongEvent = !wrongEvent

				if !include {
					break
				}
			}
		}

		if wrongEvent {
			continue
		}

		_, err = filteredEventLog.WriteString(string(line) + "\n")
		if err != nil {
			zap.S().Warnw("(SCRAPER) Failed to write line to filtered events log", "err", err)
		}
	}
}
