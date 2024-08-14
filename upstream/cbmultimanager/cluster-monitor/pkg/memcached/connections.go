// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/tools-common/errdefs"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
)

func getConnections(client *memcached.Client) ([]*values.ConnectionData, uint64, error) {
	err := client.Transmit(&gomemcached.MCRequest{
		Opcode: gomemcached.STAT,
		Opaque: 91849,
		Key:    []byte("connections "),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("could not send memcached stat request: %w", err)
	}

	var internal uint64
	connections := make([]*values.ConnectionData, 0)
	for {
		res, err := client.Receive()
		if err != nil {
			return nil, 0, fmt.Errorf("could not read memcached response: %w", err)
		}

		if len(res.Key) == 0 && len(res.Body) == 0 {
			break
		}

		var conn *values.ConnectionData
		if err = json.Unmarshal(res.Body, &conn); err != nil {
			return nil, 0, fmt.Errorf("could not unmarshal connection data: %w", err)
		}

		if conn.Internal {
			internal++
			continue
		}

		connections = append(connections, conn)
	}

	return connections, internal, nil
}

// GetConnectionsFor collects all connections to data nodes for a cluster.
func (m *MemDClient) GetConnectionsFor() (*values.ServerConnections, error) {
	connections := &values.ServerConnections{Connections: make([]*values.ConnectionData, 0)}
	errs := &errdefs.MultiError{
		Prefix: "failed to obtain connections from some nodes: ",
	}
	for _, host := range m.manager.Hosts() {
		err := m.manager.ClientForNode(host, func(client *memcached.Client) error {
			conns, internal, err := getConnections(client)
			if err != nil {
				return err
			}

			connections.InternalConnections += internal
			connections.Connections = append(connections.Connections, conns...)
			return nil
		})
		if err != nil {
			errs.Add(fmt.Errorf("could not get connection information for %s: %w", host, err))
		}
	}

	return connections, errs.ErrOrNil()
}
