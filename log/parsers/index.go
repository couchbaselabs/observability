package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// IndexCreated gets when a query index was created.
func IndexCreated(line string) (*values.Result, error) {
	if !strings.Contains(line, "OnIndexCreate") || !strings.Contains(line, "Success") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Create\sIndex\sDefnId:\s(?P<DefnId>[^\s]*)\sName:\s(?P<Name>[^\s]*)\sUsing:\s` +
		`(?P<Using>[^\s]*)\sBucket:\s(?P<Bucket>[^\s]*)\sScope\/Id:\s(?P<ScopeId>[^\s]*)\sCollection\/Id:\s` +
		`(?P<CollectionId>[^\s]*)\sIsPrimary:\s(?P<IsPrimary>[^\s]*)\sNumReplica:\s(?P<NumReplica>[^\s]*)\sInstVersion:` +
		`\s(?P<InstVersion>.*)`)
	output := lineRegexp.FindStringSubmatch(line)
	if len(output) < len(lineRegexp.SubexpNames()) {
		return nil, values.ErrRegexpMissingFields
	}

	jsonSettings := make(map[string]string)
	for i, name := range lineRegexp.SubexpNames()[1:] {
		jsonSettings[name] = output[i+1]
	}

	return &values.Result{
		Event:    values.IndexCreatedEvent,
		Settings: jsonSettings,
	}, nil
}

// IndexDeleted gets when a query index was deleted.
func IndexDeleted(line string) (*values.Result, error) {
	if !strings.Contains(line, ":OnIndexDelete") || strings.Contains(line, "Notification Received") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`IndexId\s(?P<index>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event: values.IndexDeletedEvent,
		Index: output[0][1],
	}, nil
}

// IndexerActive gets when the indexer becomes active.
func IndexerActive(line string) (*values.Result, error) {
	if !strings.Contains(line, "NewIndexer Status Active") {
		return nil, values.ErrNotInLine
	}

	return &values.Result{
		Event: values.IndexerActiveEvent,
	}, nil
}
