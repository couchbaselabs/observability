// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type IndexStatusResult struct {
	Indexes  []*values.IndexStatus `json:"indexes"`
	Version  int64                 `json:"version"`
	Warnings []string              `json:"warnings"`
}

func (c *Client) GetGSISettings() (*values.GSISettings, error) {
	res, err := c.get(GSISettingsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get GSI settings: %w", err)
	}

	var settings values.GSISettings
	if err = json.Unmarshal(res.Body, &settings); err != nil {
		return nil, fmt.Errorf("could not unmarshal GSI settings: %w", err)
	}
	return &settings, nil
}

func (c *Client) GetIndexStatus() ([]*values.IndexStatus, error) {
	// getIndexStatus is a scatter-gather endpoint, so we only need to make the request to one GSI node
	res, err := c.get(IndexStatusEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get index status: %w", err)
	}

	var result IndexStatusResult
	if err = json.Unmarshal(res.Body, &result); err != nil {
		return nil, fmt.Errorf("could not unmarshal index status: %w", err)
	}
	if len(result.Warnings) > 0 {
		zap.S().Warnw("(Couchbase) indexStatus gave us warnings", "warnings", result.Warnings)
	}
	return result.Indexes, nil
}

func (c *Client) GetIndexStorageStats() ([]*values.IndexStatsStorage, error) {
	indexesStatsStorage := make([]*values.IndexStatsStorage, 0)
	for _, node := range c.internalClient.Nodes() {
		// check if this node has the index service on it so we don't create useless clients
		if node.Services.GetPort(cbrest.ServiceGSI, c.internalClient.AltAddr()) == 0 {
			continue
		}
		// This section of code is wrapped in an anonymous function so that when each
		// iteration of the for loop is complete the resources taken up by each client
		// can be released via `defer rest.Close()`
		err := func() error {
			host, _ := node.GetQualifiedHostname(cbrest.ServiceManagement, c.internalClient.TLS(), c.internalClient.AltAddr())
			rest, err := cbrest.NewClient(cbrest.ClientOptions{
				ConnectionString: host,
				ConnectionMode:   cbrest.ConnectionModeThisNodeOnly,
				Provider: &aprov.Static{
					UserAgent: fmt.Sprintf("cbmultimanager/%s", meta.Version),
					Username:  c.authSettings.username,
					Password:  c.authSettings.password,
				},
				DisableCCP: true,
				TLSConfig:  c.authSettings.tlsConfig,
				Logger:     logger.NewToolsCommonLogger(zap.L().Sugar()),
			})
			if err != nil {
				return fmt.Errorf("could not create client for node %s: %w", host, err)
			}
			defer rest.Close()
			result, err := rest.Execute(&cbrest.Request{
				Method:             http.MethodGet,
				Endpoint:           "/stats/storage",
				Service:            cbrest.ServiceGSI,
				ExpectedStatusCode: http.StatusOK,
			})
			if err != nil {
				return fmt.Errorf("could not execute cbrest request: %w", err)
			}

			var data []*values.IndexStatsStorage
			if err = json.Unmarshal(result.Body, &data); err != nil {
				return fmt.Errorf("could not unmarshal index storage stats: %w", err)
			}
			indexesStatsStorage = append(indexesStatsStorage, data...)
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return indexesStatsStorage, nil
}
