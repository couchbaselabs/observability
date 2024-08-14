// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/log"
	"github.com/couchbase/tools-common/netutil"
)

type Client struct {
	internalClient *cbrest.Client

	ClusterInfo   *PoolsMetadata
	BootstrapTime time.Time
	authSettings  *clientAuth
}

// NewClient creates a new Couchbase REST client to use when communicating with the cluster.
func NewClient(hosts []string, user, password string, config *tls.Config, thisNodeOnly bool) (*Client, error) {
	c, err := NewUnpopulatedClientForSingleNode(hosts, user, password, config, thisNodeOnly)
	if err != nil {
		return nil, getAuthError(fmt.Errorf("could not create REST client: %w", err))
	}

	if err = c.populateClientData(user, password, config); err != nil {
		return nil, err
	}

	return c, nil
}

// populateClientData calls multiple trivial REST endpoints to populate the client object
func (c *Client) populateClientData(user, password string, config *tls.Config) error {
	c.ClusterInfo = &PoolsMetadata{
		ClusterUUID:      c.internalClient.ClusterUUID(),
		Enterprise:       c.internalClient.EnterpriseCluster(),
		DeveloperPreview: c.internalClient.DeveloperPreview(),
	}

	res, err := c.get(cbrest.EndpointPoolsDefault)
	if err != nil {
		return fmt.Errorf("error retrieveing cluster name: %w", err)
	}

	c.ClusterInfo.PoolsRaw = res.Body

	overlay := struct {
		ClusterName   string `json:"clusterName"`
		StorageTotals struct {
			HDD struct {
				QuotaTotal uint64 `json:"quotaTotal"`
				Used       uint64 `json:"used"`
				UsedByData uint64 `json:"usedByData"`
			} `json:"hdd"`
			RAM struct {
				QuotaTotal uint64 `json:"quotaTotal"`
				QuotaUsed  uint64 `json:"quotaUsed"`
			} `json:"ram"`
		} `json:"storageTotals"`
	}{}

	if err := json.Unmarshal(c.ClusterInfo.PoolsRaw, &overlay); err != nil {
		return fmt.Errorf("could not get cluster name: %w", err)
	}

	c.ClusterInfo.ClusterName = overlay.ClusterName
	c.ClusterInfo.ClusterInfo = &values.ClusterInfo{
		RAMQuota:       overlay.StorageTotals.RAM.QuotaTotal,
		RAMUsed:        overlay.StorageTotals.RAM.QuotaUsed,
		DiskTotal:      overlay.StorageTotals.HDD.QuotaTotal,
		DiskUsed:       overlay.StorageTotals.HDD.Used,
		DiskUsedByData: overlay.StorageTotals.HDD.UsedByData,
	}

	c.ClusterInfo.NodesSummary, err = c.GetNodesSummary()
	if err != nil {
		return fmt.Errorf("could not get node summary: %w", err)
	}

	c.BootstrapTime = time.Now().UTC()

	c.authSettings = &clientAuth{
		tlsConfig: config,
		username:  user,
		password:  password,
	}
	return nil
}

func (c *Client) GetBootstrap() time.Time {
	return c.BootstrapTime
}

func (c *Client) GetClusterInfo() *PoolsMetadata {
	return c.ClusterInfo
}

func NewUnpopulatedClientForSingleNode(
	hosts []string,
	user string,
	password string,
	config *tls.Config,
	thisNodeOnly bool,
) (*Client, error) {
	c := &Client{}

	opts := cbrest.ClientOptions{
		ConnectionString: netutil.HostsToConnectionString(hosts),
		Provider:         &aprov.Static{UserAgent: "cbmultimanager", Username: user, Password: password},
		TLSConfig:        config,
		ReqResLogLevel:   log.LevelDebug,
		DisableCCP:       true,
	}

	if thisNodeOnly {
		opts.ConnectionMode = cbrest.ConnectionModeThisNodeOnly
	}

	var err error
	c.internalClient, err = cbrest.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// PingNodes pings all nodes
func (c *Client) PingNodes() error {
	var errstrings []string
	hosts, serviceErr := c.internalClient.GetAllServiceHosts(cbrest.ServiceManagement)
	if serviceErr != nil {
		return fmt.Errorf("error getting management service hosts: %w", serviceErr)
	}
	for _, host := range hosts {
		request := &cbrest.Request{
			Host:     host,
			Method:   http.MethodGet,
			Endpoint: TerseClusterInfoEndpoint,
			Service:  cbrest.ServiceManagement,
		}

		_, err := c.internalClient.Execute(request)
		if err != nil {
			errstrings = append(errstrings, err.Error())
		}
	}
	if len(errstrings) > 0 {
		return fmt.Errorf("error getting terse cluster info: %v", strings.Join(errstrings, ","))
	}
	return nil
}

// getAuthError takes an error and if it is a 401/403 error it will wrap it in an AuthError. Otherwise it returns the
// error as is.
func getAuthError(err error) error {
	if err == nil {
		return nil
	}

	var boostrapError *cbrest.BootstrapFailureError
	if errors.As(err, &boostrapError) {
		if boostrapError.ErrAuthorization != (*cbrest.AuthorizationError)(nil) {
			return AuthError{err: err}
		}

		if boostrapError.ErrAuthentication != (*cbrest.AuthenticationError)(nil) {
			return AuthError{err: err, Authentication: true}
		}

		return err
	}

	var authenticationError *cbrest.AuthenticationError
	if errors.As(err, &authenticationError) {
		return AuthError{err: err, Authentication: true}
	}

	var authorizationError *cbrest.AuthorizationError
	if errors.As(err, &authorizationError) {
		return AuthError{err: err}
	}

	return err
}

func (c *Client) Close() {
	c.internalClient.Close()
}
