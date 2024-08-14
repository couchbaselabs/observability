// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package bootstrap

import (
	"fmt"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
)

type KnownCredentialsBootstrapper struct {
	username string
	password string
}

func NewKnownCredentialsBootstrapper(username, password string) *KnownCredentialsBootstrapper {
	return &KnownCredentialsBootstrapper{
		username: username,
		password: password,
	}
}

func (r *KnownCredentialsBootstrapper) CreateRESTClient() (*Node, error) {
	creds := &aprov.Static{
		UserAgent: fmt.Sprintf("cbhealthagent/%s", meta.Version),
		Username:  r.username,
		Password:  r.password,
	}
	rest, err := cbrest.NewClient(cbrest.ClientOptions{
		ConnectionString: "couchbase://localhost",
		Provider:         creds,
		ConnectionMode:   cbrest.ConnectionModeThisNodeOnly,
		Logger:           logger.NewToolsCommonLogger(zap.L().Sugar()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return prepareCluster(rest, creds)
}
