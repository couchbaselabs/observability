// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package agentport

import (
	"context"
	"fmt"
	"net/http"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/restutil"
	"github.com/couchbase/tools-common/retry"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration/tunables"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type AgentPort struct {
	logger   *zap.SugaredLogger
	client   *http.Client
	hostname string
	port     int
	creds    aprov.Provider
}

func NewAgentPort(host string, port int, creds aprov.Provider) (*AgentPort, error) {
	a := &AgentPort{
		logger:   zap.S().Named("Agent Port").With("host", host, "port", port),
		hostname: host,
		port:     port,
		client:   &http.Client{},
		creds:    creds,
	}
	if err := a.initialiseRetry(); err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}
	return a, nil
}

func (a *AgentPort) Close() error {
	return nil
}

// same as initialise but will retry
func (a *AgentPort) initialiseRetry() error {
	_, err := retry.NewRetryer(retry.RetryerOptions{MaxRetries: tunables.AgentPortMaxRetries}).
		Do(func(ctx *retry.Context) (interface{}, error) {
			return nil, a.initialise()
		})

	return err
}

func (a *AgentPort) initialise() error {
	// If the agent isn't running, these requests could block for a long time, so initialise() should log throughout,
	// otherwise cbmultimanager may look like it's hung.
	// Set a relatively low timeout for the initial ping - a ping to a healthy agent should return near-instantly
	a.logger.Infow("Initialising agent port, pinging agent.")

	var status core.PingResponse
	err := PingAgent(&status).WithTimeout(tunables.AgentPortActivateTimeout).Execute(a)
	if err != nil {
		return err
	}

	switch status.State {
	case core.AgentReady:
		return nil
	case core.AgentWaiting:
		username, password := a.creds.GetCredentials(a.hostname)
		payload := core.ActivateRequest{
			Username: username,
			Password: password,
		}

		var result string

		// activation could take longer so it has a longer timeout
		err := ActivateAgent(&payload, &result).WithTimeout(tunables.AgentPortActivateTimeout).Execute(a)
		if err != nil {
			return fmt.Errorf("failed to activate agent: %w", err)
		}

		a.logger.Infow("Agent activated", "host", a.hostname)
		return nil
	default:
		// AgentNotStarted also ends up here because an agent should never send it in reply to `ping` - by definition,
		// if it can reply to HTTP, it's started.
		return fmt.Errorf("agent in unknown state '%s'", status.State)
	}
}

func (a *AgentPort) GetCheckerResults(ctx context.Context) (map[string]values.WrappedCheckerResult, error) {
	var result map[string]values.WrappedCheckerResult

	if err := CheckerResults(&result).Execute(a); err != nil {
		return nil, fmt.Errorf("failed to get checkers: %w", err)
	}

	return result, nil
}

type AgentError struct {
	ErrorResponse restutil.ErrorResponse
}

func (a AgentError) Error() string {
	msg := fmt.Sprintf("error %d %s", a.ErrorResponse.Status, a.ErrorResponse.Msg)
	if a.ErrorResponse.Extras != "" {
		msg += fmt.Sprintf(" (%s)", a.ErrorResponse.Extras)
	}
	return msg
}
