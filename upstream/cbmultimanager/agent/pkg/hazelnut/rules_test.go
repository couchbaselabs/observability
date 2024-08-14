// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type ruleTestResult struct {
	values.CheckerResult
	RequireRemediation bool `json:"requireRemediation"`
}

type ruleTestCase struct {
	Name           string                 `json:"name"`
	Input          map[string]interface{} `json:"input"`
	ExpectedResult *ruleTestResult        `json:"expected,omitempty"`
}

type testRule struct {
	rule
	Name      string         `json:"ruleName"`
	TestCases []ruleTestCase `json:"testCases"`
}

func TestEmbeddedRulesValid(t *testing.T) {
	if _, err := loadEmbeddedRules(); err != nil {
		t.Fatalf("Embedded rules invalid: %v", err)
	}
}

func TestRules(t *testing.T) {
	var rules []testRule
	require.NoError(t, json.Unmarshal(embeddedRules, &rules))
	for _, rule := range rules {
		t.Run(rule.Name, func(t *testing.T) {
			if len(rule.TestCases) == 0 {
				t.Errorf("Rule %s has no tests!", rule.Name)
			}
			// Ignore hints, to make sure the rules still work without them
			rule.ignoreHints = true
			for _, tc := range rule.TestCases {
				t.Run(tc.Name, func(t *testing.T) {
					result, err := rule.apply(time.Now(), tc.Input)
					require.NoError(t, err)

					if tc.ExpectedResult == nil {
						require.Nil(t, result, "got a match when we weren't expecting one")
						return
					}
					require.Equal(t, tc.ExpectedResult.Name, result.Name, "checker names do not match")
					require.Equal(t, tc.ExpectedResult.Status, result.Status, "checker statuses do not match")
					if tc.ExpectedResult.RequireRemediation {
						require.NotEmpty(t, result.Remediation, "expected remediation but didn't get one")
					} else {
						require.Empty(t, result.Remediation, "got remediation when not expecting one")
					}
					if tc.ExpectedResult.Remediation != "" {
						require.Equal(t, tc.ExpectedResult.Remediation, result.Remediation, "remediations do not match")
					}
				})
			}
		})
	}
}
