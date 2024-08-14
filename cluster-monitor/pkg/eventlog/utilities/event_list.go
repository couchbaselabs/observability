// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package utilities

import (
	"fmt"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

func GetEventList(eventString string) ([]values.EventType, error) {
	if eventString == "" {
		return nil, nil
	}

	stringList := strings.Split(eventString, ",")
	eventMap := make(map[values.EventType]struct{})
	var events []values.EventType

	for _, stringEvent := range stringList {
		if _, ok := values.EventTypes[stringEvent]; !ok {
			return nil, fmt.Errorf("invalid event type given: %s", stringEvent)
		}

		if _, ok := eventMap[values.EventTypes[stringEvent]]; !ok {
			events = append(events, values.EventTypes[stringEvent])
			eventMap[values.EventTypes[stringEvent]] = struct{}{}
		}
	}

	return events, nil
}
