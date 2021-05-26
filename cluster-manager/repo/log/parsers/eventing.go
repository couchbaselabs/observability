package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// EventFunctionDeployedOrUndeployed gets when an eventing function is deployed or undeployed.
func EventFunctionDeployedOrUndeployed(line string) (*values.Result, error) {
	if !strings.Contains(line, "setSettings") || !strings.Contains(line, "params:") ||
		!strings.Contains(line, "deployment_status") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Function:\s(?P<function>[^\s]*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	event := values.EventingFunctionUndeployedEvent
	if strings.Contains(line, "deployment_status:true") {
		event = values.EventingFunctionDeployedEvent
	}

	return &values.Result{
		Event:    event,
		Function: output[0][1],
	}, nil
}
