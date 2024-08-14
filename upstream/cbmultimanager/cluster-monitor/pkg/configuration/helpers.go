// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package configuration

import (
	"fmt"
	"strings"
)

func ParseLabelSelectors(input string) (LabelSelectors, error) {
	result := make(LabelSelectors)
	if len(input) == 0 {
		return result, nil
	}
	parts := strings.Split(input, " ")
	for _, part := range parts {
		chunks := strings.SplitN(part, "=", 2)
		if len(chunks) != 2 {
			return nil, fmt.Errorf("parse error for label selector '%s'", part)
		}
		result[chunks[0]] = chunks[1]
	}
	return result, nil
}
