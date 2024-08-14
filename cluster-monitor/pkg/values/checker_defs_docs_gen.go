// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

//go:build gen
// +build gen

package values

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type CheckerArea string

const (
	ClusterChecker CheckerArea = "cluster"
	NodeChecker    CheckerArea = "node"
	BucketChecker  CheckerArea = "bucket"
)

type CheckerDefWithDoc struct {
	CheckerDefinition         `yaml:",inline"`
	Area                      CheckerArea `yaml:"area" json:"-"`
	Agent                     bool        `yaml:"agent" json:"-"`
	Background                string      `yaml:"background" json:"-"`
	Condition                 string      `yaml:"condition" json:"-"`
	Remediation               string      `yaml:"remediation" json:"-"`
	FurtherReading            string      `yaml:"further_reading" json:"-"`
	DocumentedInObservability bool        `yaml:"documented_in_observability" json:"-"`
}

func LoadCheckerDefsWithDocs() map[string]CheckerDefWithDoc {
	var result map[string]CheckerDefWithDoc
	if err := yaml.Unmarshal(checkerDefsData, &result); err != nil {
		panic(fmt.Errorf("failed to load checker defs w/ docs: %w", err))
	}
	return result
}
