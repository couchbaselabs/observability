// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"crypto/tls"
	"fmt"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	"github.com/couchbase/tools-common/netutil"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
)

//go:generate mockery --name mcClientFactory --exported

// mcClientFactory creates a memcached client.
// It is an interface to make testing easier.
type mcClientFactory interface {
	CreateClient(host, username, password string, config *tls.Config) (*memcached.Client, error)
}

type defaultMCClientFactory struct{}

func (defaultMCClientFactory) CreateClient(host, user, password string, config *tls.Config) (
	*memcached.Client, error,
) {
	var (
		client *memcached.Client
		err    error
	)

	if config != nil {
		client, err = memcached.ConnectTLS("tcp", netutil.TrimSchema(host), config)
	} else {
		client, err = memcached.Connect("tcp", netutil.TrimSchema(host))
	}

	if err != nil {
		return nil, fmt.Errorf("cannot create client: %w", err)
	}

	_, err = client.Auth(user, password)
	if err != nil {
		return nil, fmt.Errorf("cannot auth against cluster: %w", err)
	}

	_, err = client.Send(&gomemcached.MCRequest{
		Opcode: gomemcached.HELLO,
		Key:    []byte(fmt.Sprintf("cbmultimanager/v%s", meta.Version)),
	})

	if err != nil {
		return nil, fmt.Errorf("could not HELLO server: %w", err)
	}

	return client, nil
}
