// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package agentport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
)

const (
	ContentTypeURLEncoded string = "application/x-www-form-urlencoded"
	ContentTypeJSON       string = "application/json"
	ContentTypePlain      string = "text/plain"

	HeaderContentType string = "Content-Type"
	HeaderUserAgent   string = "User-Agent"
)

func (a *AgentPort) doRequest(request *http.Request, result interface{}, revive bool) error {
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// check the header
		if revive && response.Header.Get(core.AgentActiveHeader) == "false" {
			a.logger.Warn("Agent is inactivate")
			err := a.initialiseRetry()
			if err != nil {
				return fmt.Errorf("received status %d, tried to reactivate but failed: %w", response.StatusCode, err)
			}

			// successfully revived the agent so lets try again
			return a.doRequest(request, result, revive)
		}
		return fmt.Errorf("request failed %v %v %v: %v", request.Method, request.URL.String(), response.Status, string(body))
	}

	if result == nil {
		return nil
	}

	// Handle the content types we expect.
	contentType := response.Header.Get(HeaderContentType)
	switch {
	case strings.Contains(contentType, ContentTypeJSON):
		if err := json.Unmarshal(body, result); err != nil {
			return err
		}
	case strings.Contains(contentType, ContentTypePlain):
		s, ok := result.(*string)
		if !ok {
			return fmt.Errorf("%w: unexpected type decode for text/plain", err)
		}

		*s = string(body)
	default:
		return fmt.Errorf("unexpected content type %s", contentType)
	}

	return nil
}

// appends path to the agent base url
func (a *AgentPort) makeURL(path string) string {
	return fmt.Sprintf("http://%s:%d%s", a.hostname, a.port, path)
}

func (a *AgentPort) Get(ctx context.Context, r *Request) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.makeURL(r.Path), nil)
	if err != nil {
		return err
	}

	a.setDefaultHeaders(req)

	return a.doRequest(req, r.Result, r.revive)
}

func (a *AgentPort) Post(ctx context.Context, r *Request) error {
	req, err := http.NewRequestWithContext(ctx, "POST", a.makeURL(r.Path), bytes.NewReader(r.Body))
	if err != nil {
		return err
	}

	a.setDefaultHeaders(req)
	req.Header.Set(HeaderContentType, ContentTypeJSON)

	return a.doRequest(req, r.Result, r.revive)
}

func (a *AgentPort) setDefaultHeaders(r *http.Request) {
	headers := http.Header{}
	headers.Set(HeaderUserAgent, fmt.Sprintf("cbmultimanager/%s agent-port", meta.Version))
	r.Header = headers
}
