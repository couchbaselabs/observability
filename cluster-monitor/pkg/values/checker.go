// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/tools-common/cbvalue"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// CheckerType is used to differentiate by checkers that work on different type of information.
type CheckerType uint16

func (c *CheckerType) UnmarshalYAML(value *yaml.Node) error {
	var strVal string
	if err := value.Decode(&strVal); err != nil {
		return err
	}
	return c.UnmarshalText([]byte(strVal))
}

const (
	APICheckerType CheckerType = iota
	LogCheckerType
	SystemCheckType
)

// UnmarshalText parses the string version of this checker type.
func (c *CheckerType) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "api":
		*c = APICheckerType
		return nil
	case "log":
		*c = LogCheckerType
		return nil
	case "system":
		*c = SystemCheckType
		return nil
	default:
		return fmt.Errorf("unknown checker type '%s'", text)
	}
}

func (c CheckerType) MarshalText() (text []byte, err error) {
	switch c {
	case APICheckerType:
		return []byte("api"), nil
	case LogCheckerType:
		return []byte("log"), nil
	case SystemCheckType:
		return []byte("system"), nil
	default:
		return nil, fmt.Errorf("invalid checker type %v", c)
	}
}

func (c CheckerType) MarshalJSON() ([]byte, error) {
	strVal, err := c.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(strVal))
}

// CheckerStatus is an alias to use for the different possible checker statuses
type CheckerStatus string

func (c CheckerStatus) Int() int {
	switch c {
	case GoodCheckerStatus:
		return 0
	case InfoCheckerStatus:
		return 10
	case WarnCheckerStatus:
		return 20
	case AlertCheckerStatus:
		return 30
	case MissingCheckerStatus:
		return -1
	default:
		return -1
	}
}

func (c CheckerStatus) Severity() string {
	switch c {
	case InfoCheckerStatus:
		return "info"
	case WarnCheckerStatus:
		return "warning"
	case AlertCheckerStatus:
		return "critical"
	case GoodCheckerStatus, MissingCheckerStatus:
		// By definition, there isn't one
		fallthrough
	default:
		zap.S().Warnw("(Values) Tried to convert invalid CheckerStatus to severity", "status", c)
		return ""
	}
}

const (
	GoodCheckerStatus    CheckerStatus = "good"
	WarnCheckerStatus    CheckerStatus = "warn"
	AlertCheckerStatus   CheckerStatus = "alert"
	InfoCheckerStatus    CheckerStatus = "info"
	MissingCheckerStatus CheckerStatus = "missing"
)

// CheckerDefinition is used to map a checker name to information to display in the UI about the checker. This avoids
// storing duplicate information in the store.
type CheckerDefinition struct {
	Name        string          `json:"name" yaml:"name"`
	ID          string          `json:"id" yaml:"id"`
	Title       string          `json:"title" yaml:"title"`
	Type        CheckerType     `json:"type" yaml:"type"`
	Description string          `json:"description" yaml:"short_description"` // backwards-compat
	MinVersion  cbvalue.Version `json:"min_version,omitempty" yaml:"min_version,omitempty"`
	MaxVersion  cbvalue.Version `json:"max_version,omitempty" yaml:"max_version,omitempty"`
}

// CheckerResult is a structure that contains what is expected for a checker function to return.
type CheckerResult struct {
	Name        string          `json:"name"`
	Remediation string          `json:"remediation,omitempty"`
	Value       json.RawMessage `json:"value,omitempty"`
	Status      CheckerStatus   `json:"status"`
	Time        time.Time       `json:"time"`

	Version int `json:"version"`
}

func (c *CheckerResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name        string          `json:"name"`
		Remediation string          `json:"remediation,omitempty"`
		Value       json.RawMessage `json:"value,omitempty"`
		Status      CheckerStatus   `json:"status"`
		StatusCode  int             `json:"status_code"`
		Time        time.Time       `json:"time"`

		Version int `json:"version"`
	}{
		Name:        c.Name,
		Remediation: c.Remediation,
		Value:       c.Value,
		StatusCode:  c.Status.Int(),
		Status:      c.Status,
		Time:        c.Time,
		Version:     c.Version,
	})
}

// WrappedCheckerResult will be used so that checkers can return multiple results for different containers as well
// as independent errors.
type WrappedCheckerResult struct {
	Result  *CheckerResult `json:"result"`
	Error   error          `json:"-"`
	Cluster string         `json:"cluster"`
	Bucket  string         `json:"bucket,omitempty"`
	Node    string         `json:"node,omitempty"`
	LogFile string         `json:"log_file,omitempty"`
}

// CheckerFn perform one or more arbitrary checks on a given cluster.
type CheckerFn func(cluster CouchbaseCluster) ([]*WrappedCheckerResult, error)

// CheckerSearch is used as a parameter to search for the correct level of checkers.
type CheckerSearch struct {
	Name    *string
	Cluster *string
	Node    *string
	LogFile *string
	Bucket  *string
}
