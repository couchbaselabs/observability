package connections

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/netutil"

	"go.uber.org/zap"
)

func getMemcachedClient(host, user, password string, config *tls.Config) (*memcached.Client, error) {
	var (
		client *memcached.Client
		err    error
	)

	if config != nil {
		client, err = memcached.ConnectTLS("tcp", host, config)
	} else {
		client, err = memcached.Connect("tcp", host)
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
		Key:    []byte("cbmultimanager/v0.0.0"),
	})
	if err != nil {
		return nil, fmt.Errorf("could not HELLO server: %w", err)
	}

	return client, nil
}

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

func GetConnectionsFor(cluster *values.CouchbaseCluster) (*values.ServerConnections, error) {
	start := time.Now()
	zap.S().Infow("(Connection) Getting connections for cluster", "cluster", cluster.UUID)

	restClient, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("could not create the rest client")
	}

	kvHosts, err := restClient.GetAllServiceHosts(cbrest.ServiceData)
	if err != nil {
		return nil, fmt.Errorf("could not get data nodes: %w", err)
	}

	if len(kvHosts) == 0 {
		return nil, fmt.Errorf("could not retrieve kv hosts")
	}

	connections := &values.ServerConnections{Connections: make([]*values.ConnectionData, 0)}

	getConnections := func(host string) ([]*values.ConnectionData, uint64, error) {
		client, err := getMemcachedClient(host, cluster.User, cluster.Password, cluster.GetTLSConfig())
		if err != nil {
			return nil, 0, err
		}

		defer client.Close()

		return getConnections(client)
	}

	for _, host := range kvHosts {
		noSchemeHost := netutil.TrimSchema(host)
		conns, internalCount, err := getConnections(noSchemeHost)
		if err != nil {
			return nil, fmt.Errorf("could not get connections for host '%s': %w", noSchemeHost, err)
		}

		connections.InternalConnections += internalCount
		connections.Connections = append(connections.Connections, conns...)
	}

	zap.S().Debugw("(Connection) Got all connections for cluster", "cluster", cluster.UUID, "elapsed",
		time.Since(start).String())
	return connections, nil
}
