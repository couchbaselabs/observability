// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package alertmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager/types"
)

//go:generate mockery --name alertmanagerClientIFace --exported

// alertmanagerClientIFace is present to make testing AlertGenerator easier.
type alertmanagerClientIFace interface {
	PostAlerts(ctx context.Context, alerts []types.PostableAlert) error
	BaseURL() string
}

type alertmanagerClient struct {
	httpClient *http.Client
	baseURL    string
}

func newAlertmanagerClient(baseURL string) *alertmanagerClient {
	client := http.Client{
		Timeout: time.Minute,
	}
	return &alertmanagerClient{
		httpClient: &client,
		baseURL:    baseURL,
	}
}

func (a *alertmanagerClient) BaseURL() string {
	return a.baseURL
}

const apiV2AlertsPath = "/api/v2/alerts"

func (a *alertmanagerClient) PostAlerts(ctx context.Context, alerts []types.PostableAlert) error {
	payload, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+apiV2AlertsPath, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(res.Body)
	var bodyStr string

	if strings.Contains(res.Header.Get("Content-Type"), "application/json") ||
		strings.Contains(res.Header.Get("Content-Type"), "text/") {
		bodyStr = string(body)
	} else {
		bodyStr = fmt.Sprintf("%v", body)
	}
	return fmt.Errorf("got %d status: %s", res.StatusCode, bodyStr)
}

type multiError []error

func (e *multiError) add(err error) {
	if err != nil {
		*e = append(*e, err)
	}
}

func (e multiError) Error() string {
	errStr := strings.Builder{}
	for _, err := range e[:len(e)-1] {
		errStr.WriteString(err.Error())
		errStr.WriteRune('\n')
	}
	errStr.WriteString(e[len(e)-1].Error())
	return errStr.String()
}
