package parsers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// BucketCreated gets when a bucket was created along with all of the config parameters.
func BucketCreated(line string) (*values.Result, error) {
	if !strings.Contains(line, "do_ensure_bucket") || !strings.Contains(line, "Created bucket") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Created\sbucket\s"(?P<bucket>[^"]*)".*bucket_type=(?P<bucket_type>[^;]*);`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:      values.BucketCreatedEvent,
		Bucket:     output[0][1],
		BucketType: output[0][2],
	}, nil
}

// BucketDeleted gets when a bucket was deleted.
func BucketDeleted(line string) (*values.Result, error) {
	if !strings.Contains(line, "Deleted bucket") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Deleted\sbucket\s"(?P<bucket>[^"]*)"`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:  values.BucketDeletedEvent,
		Bucket: output[0][1],
	}, nil
}

// BucketUpdated gets when a bucket was updated.
func BucketUpdated(line string) (*values.Result, error) {
	if !strings.Contains(line, "Updated bucket") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Updated\sbucket\s"(?P<bucket>[^"]*)".*properties:\[(?P<config>[^\]]*)\]`)
	bracketRegexp := regexp.MustCompile(`\{[^\}]*\}`)

	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	// convert format: [{num_replicas,0},{ram_quota,209715200},{flush_enabled,false},{storage_mode,couchstore}]
	// into format: map[string]string{"num_replicas":  "0", "ram_quota": "209715200", "flush_enabled": "false",
	//		"storage_mode":  "couchstore"}
	config := output[0][2]
	// get a list of settings
	configList := bracketRegexp.FindAllString(config, -1)
	settings := make(map[string]string)

	for _, setting := range configList {
		// remove brackets
		setting = setting[1 : len(setting)-1]
		// split name from value
		settingSlice := strings.Split(setting, ",")
		if len(settingSlice) < 2 {
			return nil, errors.New("missing config parameter value")
		}

		// add setting to map
		settings[settingSlice[0]] = settingSlice[1]
	}

	return &values.Result{
		Event:    values.BucketUpdatedEvent,
		Bucket:   output[0][1],
		Settings: settings,
	}, nil
}

// BucketFlushed gets when a bucket was flushed.
func BucketFlushed(line string) (*values.Result, error) {
	if !strings.Contains(line, "Flushing bucket") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Flushing\sbucket\s"(?P<bucket>[^"]*)".*node\s'(?P<node>[^']*)'`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:  values.BucketFlushedEvent,
		Bucket: output[0][1],
		Node:   output[0][2],
	}, nil
}
