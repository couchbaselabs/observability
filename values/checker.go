package values

import (
	"encoding/json"
	"time"
)

// CheckerType is used to differentiate by checkers that work on different type of information.
type CheckerType uint16

const (
	APICheckerType CheckerType = iota
	LogCheckerType
)

// Status is an alias to use for the different possible checker statuses
type CheckerStatus string

func (c CheckerStatus) Int() int {
	switch c {
	case GoodCheckerStatus:
		return 0
	case WarnCheckerStatus:
		return 1
	case AlertCheckerStatus:
		return 2
	case InfoCheckerStatus:
		return 3
	case MissingCheckerStatus:
		return 4
	default:
		return -1
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
	Name        string      `json:"name"`
	Title       string      `json:"title"`
	Type        CheckerType `json:"type"`
	Description string      `json:"description"`
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
type CheckerFn func(cluster *CouchbaseCluster) ([]*WrappedCheckerResult, error)

// CheckerSearch is used as a parameter to search for the correct level of checkers.
type CheckerSearch struct {
	Name    *string
	Cluster *string
	Node    *string
	LogFile *string
	Bucket  *string
}
