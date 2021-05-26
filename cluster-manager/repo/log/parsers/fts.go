package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// FTSIndexCreatedOrDropped gets when a full-text index is created or dropped.
func FTSIndexCreatedOrDropped(line string) (*values.Result, error) {
	if !strings.Contains(line, "index definition created") && !strings.Contains(line, "index definition deleted") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`indexName:\s(?P<index>[^,]*),`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	event := values.FTSIndexDroppedEvent
	if strings.Contains(line, "index definition created") {
		event = values.FTSIndexCreatedEvent
	}

	return &values.Result{
		Event: event,
		Index: output[0][1],
	}, nil
}
