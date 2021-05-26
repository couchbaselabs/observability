package couchbase

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/log"
	"github.com/couchbase/tools-common/netutil"
)

type Client struct {
	internalClient *cbrest.Client

	ClusterInfo   *PoolsMetadata
	BootstrapTime time.Time
}

// NewClient creates a new Couchbase REST client to use when communicating with the cluster.
func NewClient(hosts []string, user, password string, config *tls.Config) (*Client, error) {
	c := &Client{}

	var err error
	c.internalClient, err = cbrest.NewClient(cbrest.ClientOptions{
		ConnectionString: netutil.HostsToConnectionString(hosts),
		Username:         user,
		Password:         password,
		UserAgent:        "cbmultimanager",
		TLSConfig:        config,
		ReqResLogLevel:   log.LevelDebug,
	})
	if err != nil {
		return nil, getAuthError(fmt.Errorf("could not create REST client: %w", err))
	}

	c.ClusterInfo = &PoolsMetadata{
		ClusterUUID: c.internalClient.ClusterUUID(),
	}

	res, err := c.get(cbrest.EndpointPoolsDefault)
	if err != nil {
		return nil, fmt.Errorf("error retrieveing cluster name: %w", err)
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
		return nil, fmt.Errorf("could not get cluster name: %w", err)
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
		return nil, fmt.Errorf("could not get node summary: %w", err)
	}

	c.BootstrapTime = time.Now().UTC()

	return c, nil
}

func (c *Client) GetBootstrap() time.Time {
	return c.BootstrapTime
}

func (c *Client) GetClusterInfo() *PoolsMetadata {
	return c.ClusterInfo
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
