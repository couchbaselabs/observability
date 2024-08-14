// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// stringRegexp wraps Regexp to parse a regular expression from JSON.
type stringRegexp struct {
	*regexp.Regexp
}

type customFields []customField

type customField struct {
	FieldName     string       `json:"name"`
	Type          string       `json:"type"` // time and string
	Regexp        stringRegexp `json:"regexp"`
	TimeThreshold string       `json:"timeThreshold"` // Maybe a better name exists.
}

func (s *stringRegexp) UnmarshalJSON(i []byte) error {
	var strVal string
	err := json.Unmarshal(i, &strVal)
	if err != nil {
		return err
	}
	s.Regexp, err = regexp.Compile(strVal)
	if err != nil {
		return fmt.Errorf("invalid regexp: %w", err)
	}
	return nil
}

// rule represents a logging rule.
type rule struct {
	File         string                 `json:"file"`
	Contains     string                 `json:"contains"`
	Regexp       stringRegexp           `json:"regexp"`
	Result       values.CheckerResult   `json:"result"`
	Hints        map[string]interface{} `json:"hints"`
	CustomFields customFields           `json:"customFields"`
	// Only used for testing
	ignoreHints bool
}

// applyCustom enforces all custom fields provided in JSON.
func (r *rule) applyCustom(packet map[string]interface{}) (bool, error) {
	if r.CustomFields == nil {
		return true, nil
	}
	for _, customRule := range r.CustomFields {
		if _, ok := packet[customRule.FieldName]; !ok {
			return false, fmt.Errorf("invalid input: no custom field(%v) in log", customRule.FieldName)
		}
		customRuleData, ok := packet[customRule.FieldName].(string)
		if !ok {
			return false, fmt.Errorf("invalid input: custom field(%v) data is not a string", customRule.FieldName)
		}

		switch customRule.Type {
		case "string":
			if customRule.Regexp.Regexp != nil {
				customMatch := customRule.Regexp.FindString(customRuleData)
				if customMatch == "" {
					return false, nil
				}
			} else {
				return false, fmt.Errorf("no regex provided for type string in custom field(%v)", customRule.FieldName)
			}
		case "time":
			if customRule.TimeThreshold != "" {
				timeBreach, err := time.ParseDuration(customRule.TimeThreshold)
				if err != nil {
					return false, fmt.Errorf("unable to parse time breach given in custom field(%v)", customRule.FieldName)
				}
				timeFromLog, err := time.ParseDuration(customRuleData)
				if err != nil {
					return false, fmt.Errorf("unable to parse time in log for custom field(%v)", customRule.FieldName)
				}
				if timeFromLog < timeBreach {
					return false, nil
				}
			}
		case "default":
			return false, fmt.Errorf("type field incorrect for custom field(%v)", customRule.FieldName)
		}
	}
	return true, nil
}

func (r *rule) apply(ts time.Time, packet map[string]interface{}) (*values.CheckerResult, error) {
	file, ok := packet["file"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: file is not a string")
	}
	if file != r.File {
		return nil, nil
	}

	if !r.ignoreHints {
		for key, val := range r.Hints {
			if test, ok := packet[key]; !ok || test != val {
				return nil, nil
			}
		}
	}

	if _, ok := packet["message"]; !ok {
		return nil, fmt.Errorf("invalid input: no message")
	}
	message, ok := packet["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: message is not a string")
	}

	if r.Contains != "" && !strings.Contains(message, r.Contains) {
		return nil, nil
	}

	var match []int
	// NOTE: it is valid for a rule to not have a regexp.
	if r.Regexp.Regexp != nil {
		match = r.Regexp.FindStringSubmatchIndex(message)
		if match == nil {
			return nil, nil
		}
	}

	// NOTE: Custom field with type `string` should have regexp.
	// Custom field with type `time` should have timeThreshold.
	customsMatch, err := r.applyCustom(packet)
	if err != nil {
		return nil, err
	}
	if !customsMatch {
		return nil, nil
	}

	// We've got a match. Copy the Result and apply templates to its remediation.
	// If the rule doesn't have a regexp, there's no templating to be done, so just apply it verbatim.
	result := r.Result
	if r.Regexp.Regexp != nil {
		buffer := make([]byte, 0)
		result.Remediation = string(r.Regexp.ExpandString(buffer, result.Remediation, message, match))
	}
	result.Time = ts
	return &result, nil
}

func (r *Receiver) processMessages(reader io.Reader) error {
	// Read the first byte to determine the type of the data stream
	// If it's 0x7B ('{'), it's almost certainly JSON (technically valid msgpack, but we should never get raw fixints)
	// Otherwise, assume it's msgpack
	tmpBuf := make([]byte, 1)
	_, err := reader.Read(tmpBuf)
	if err != nil {
		return fmt.Errorf("error when reading first byte: %w", err)
	}
	// Re-assemble the reader including the first byte
	rdr := io.MultiReader(bytes.NewReader(tmpBuf), reader)
	var decoder messageDecoder
	if tmpBuf[0] == '{' {
		decoder = &jsonDecoder{dec: json.NewDecoder(rdr)}
	} else {
		decoder = &msgpackDecoder{dec: msgpack.NewDecoder(rdr)}
	}
	for {
		ts, data, err := decoder.Decode()
		if err != nil {
			return fmt.Errorf("failed to decode log: %w", err)
		}
		r.logger.Debug("Got some data", zap.Any("raw", data))

		if _, ok := data["file"]; !ok {
			r.logger.Warn("Invalid data: no file")
			continue
		}
		file, ok := data["file"].(string)
		if !ok {
			r.logger.Warn("Invalid data: file is not a string")
			continue
		}

		for _, rule := range r.fileRules[file] {
			result, err := rule.apply(ts, data)
			if err != nil {
				r.logger.Warn("Error occurred while applying rule", zap.Error(err))
				continue
			}
			if result != nil {
				r.resultStore.SetCheckerResult(result.Name, &values.WrappedCheckerResult{
					Result:  result,
					Node:    r.node.UUID(),
					Cluster: r.node.ClusterUUID(),
				})
			}
		}
	}
}
