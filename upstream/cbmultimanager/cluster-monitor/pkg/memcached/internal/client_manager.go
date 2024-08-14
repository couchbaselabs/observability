// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"crypto/tls"
	"fmt"
	"sync"

	memcached "github.com/couchbase/gomemcached/client"
	"github.com/couchbase/tools-common/errdefs"
	"go.uber.org/zap"
)

type mcClientMap map[string]*memcached.Client

type ClientManager struct {
	hosts     []string
	user      string
	pasword   string
	tlsConfig *tls.Config

	factory mcClientFactory

	clients    mcClientMap
	clientsMux sync.Mutex
}

func NewClientManager(hosts []string, user, password string, cfg *tls.Config) *ClientManager {
	return &ClientManager{
		hosts:     hosts,
		user:      user,
		pasword:   password,
		tlsConfig: cfg,
		factory:   defaultMCClientFactory{},
		clients:   make(mcClientMap),
	}
}

// Hosts returns the host-port strings of all KV servers known to this ClientManager.
func (m *ClientManager) Hosts() []string {
	return m.hosts
}

// ClientForNode calls the given fn with a memcached client connected to the given node, creating one if necessary.
//
// fn is guaranteed to be executed synchronously (i.e. ClientForNode will not return until fn does), and ClientForNode
// will ensure that no other caller can use the client until fn returns. However, it is not safe to continue using the
// client after fn returns.
//
// ClientForNode will return an error if it fails to create a client, or if fn returns one (in which case it will be
// returned verbatim).
func (m *ClientManager) ClientForNode(host string, fn func(client *memcached.Client) error) error {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()

	client, ok := m.clients[host]
	if !ok {
		zap.S().Debugw("(Memcached) Creating new client", "host", host)
		var err error
		client, err = m.factory.CreateClient(host, m.user, m.pasword, m.tlsConfig)
		if err != nil {
			return fmt.Errorf("could not create memcached client for %s: %w", host, err)
		}
		m.clients[host] = client
	}

	return fn(client)
}

// Close shuts down all memcached clients that this client manager has open, returning an error if any of their
// Close() methods return an error.
func (m *ClientManager) Close() error {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	zap.S().Debugw("(Memcached) Shutting down all clients", "n", len(m.clients))

	errs := new(errdefs.MultiError)
	for _, client := range m.clients {
		errs.Add(client.Close())
	}
	return errs.ErrOrNil()
}
