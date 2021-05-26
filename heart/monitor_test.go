package heart

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/storage/sqlite"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/netutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.ConsoleSeparator = " "

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, os.Stdout, zapcore.WarnLevel)

	zap.ReplaceGlobals(zap.New(core))
}

// TestHeartMonitorClusterOK will create a store a cluster and respond to the client heartbeat correctly. After some
// heartbeats it will stop the monitor and check that the cluster has been updated in the store.
func TestHeartMonitorClusterOK(t *testing.T) {
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store.sqlite"), "key")
	require.NoError(t, err)
	defer store.Close()

	testHandler := couchbase.TestHandler{
		PoolsDefault: couchbase.TestPoolsDefaultData{
			ClusterName:   "test-cluster",
			StorageTotals: couchbase.TestStorageTotals{},
			Nodes: []struct {
				Version string `json:"version"`
			}{
				{
					Version: "7.0.0-0000-enterprise",
				},
			},
		},
		ClusterUUID: "uuid-0",
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "N0",
				Services:          []string{"kv"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports: map[string]uint16{
					"httpsMgmt": 9000,
				},
			},
		},
		Buckets:          []couchbase.BucketsEndpointData{},
		NodesReturnCode:  http.StatusOK,
		BucketReturnCode: http.StatusOK,
	}

	testHandler.Start(t, true)
	defer testHandler.Close()

	noSchemaHost := netutil.TrimSchema(testHandler.URL())
	_, port, err := net.SplitHostPort(noSchemaHost)
	require.NoError(t, err, "could not split hosts ports")

	portNum, err := strconv.Atoi(port)
	require.NoError(t, err, "invalid port")

	testHandler.Nodes = []couchbase.TestNode{
		{
			NodeUUID:          "N0",
			Hostname:          noSchemaHost,
			Services:          []string{"kv"},
			Version:           "7.0.0-0000-enterprise",
			Status:            "healthy",
			ClusterMembership: "active",
			Ports: map[string]uint16{
				"httpsMgmt": uint16(portNum),
			},
		},
	}

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "cluster-0",
		User:     "user",
		Password: "password",
		BucketsSummary: values.BucketsSummary{
			{
				Name: "B0",
			},
		},
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "N0",
				Version:           "7.0.0-0000-enterprise",
				Host:              testHandler.URL(),
				Status:            "warmup",
				ClusterMembership: "active",
				Services:          []string{"kv", "backup"},
			},
		},
		ClusterInfo: &values.ClusterInfo{},
	}

	err = store.AddCluster(cluster)
	require.NoError(t, err, "could not add test cluster")

	beforeHeartBeat := time.Now()

	monitor := NewMonitor(store, 1)
	monitor.Start(300 * time.Millisecond)
	time.Sleep(1 * time.Second)
	monitor.Stop()

	cluster, err = store.GetCluster("uuid-0", true)
	require.NoError(t, err)

	require.Falsef(t, cluster.LastUpdate.Before(beforeHeartBeat), "Expected the last update to be after %v got %v",
		beforeHeartBeat, cluster.LastUpdate)

	require.Len(t, cluster.BucketsSummary, 0)

	//  things we do not want to compare
	cluster.LastUpdate = time.Time{}
	cluster.ClusterInfo = nil
	cluster.BucketsSummary = nil
	cluster.CaCert = nil

	expectedCluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "test-cluster",
		User:     "user",
		Password: "password",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "N0",
				Version:           "7.0.0-0000-enterprise",
				Host:              testHandler.URL(),
				Status:            "healthy",
				ClusterMembership: "active",
				Services:          []string{"kv"},
			},
		},
		HeartBeatIssue: values.NoHeartIssue,
	}

	require.Equal(t, expectedCluster, cluster)
}

func TestHeartMonitorClusterBadAuth(t *testing.T) {
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store-1.sqlite"), "key")
	require.NoError(t, err)
	defer store.Close()

	testHandler := couchbase.TestHandler{
		ClusterUUID: "uuid-0",
		PoolsDefault: couchbase.TestPoolsDefaultData{
			ClusterName:   "test-cluster",
			StorageTotals: couchbase.TestStorageTotals{},
			Nodes: []struct {
				Version string `json:"version"`
			}{
				{
					Version: "7.0.0-0000-enterprise",
				},
			},
		},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "N0",
				Services:          []string{"kv"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports: map[string]uint16{
					"httpsMgmt": 9000,
				},
			},
		},
		Buckets:          []couchbase.BucketsEndpointData{},
		NodesReturnCode:  http.StatusUnauthorized,
		BucketReturnCode: http.StatusUnauthorized,
	}

	testHandler.Start(t, true)
	defer testHandler.Close()

	noSchemaHost := netutil.TrimSchema(testHandler.URL())
	_, port, err := net.SplitHostPort(noSchemaHost)
	require.NoError(t, err, "could not split hosts ports")

	portNum, err := strconv.Atoi(port)
	require.NoError(t, err, "invalid port")

	testHandler.Nodes = []couchbase.TestNode{
		{
			NodeUUID:          "N0",
			Hostname:          noSchemaHost,
			Services:          []string{"kv"},
			Version:           "7.0.0-0000-enterprise",
			Status:            "healthy",
			ClusterMembership: "active",
			Ports: map[string]uint16{
				"httpsMgmt": uint16(portNum),
			},
		},
	}

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "cluster-0",
		User:     "user",
		Password: "password",
		BucketsSummary: values.BucketsSummary{
			{
				Name: "B0",
			},
		},
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "N0",
				Version:           "7.0.0-0000-enterprise",
				Host:              testHandler.URL(),
				Status:            "warmup",
				ClusterMembership: "active",
				Services:          []string{"kv", "backup"},
			},
		},
		ClusterInfo: &values.ClusterInfo{},
	}

	err = store.AddCluster(cluster)
	require.NoError(t, err, "could not add test cluster")

	beforeHeartBeat := time.Now()

	monitor := NewMonitor(store, 1)
	monitor.Start(300 * time.Millisecond)
	time.Sleep(1 * time.Second)
	monitor.Stop()

	outCluster, err := store.GetCluster("uuid-0", true)
	require.NoError(t, err)

	require.Falsef(t, outCluster.LastUpdate.Before(beforeHeartBeat), "Expected the last update to be after %v got %v",
		beforeHeartBeat, outCluster.LastUpdate)

	//  things we do not want to compare
	outCluster.LastUpdate = time.Time{}
	outCluster.ClusterInfo = nil
	outCluster.CaCert = nil

	cluster.ClusterInfo = nil
	cluster.CaCert = nil
	cluster.LastUpdate = time.Time{}
	cluster.HeartBeatIssue = values.BadAuthHeartIssue

	require.Equal(t, cluster, outCluster)
}

func TestHeartMonitorClusterUUIDMismatch(t *testing.T) {
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store.sqlite"), "key")
	if err != nil {
		t.Fatalf("Could nt initialize store for test: %v", err)
	}

	testHandler := couchbase.TestHandler{
		ClusterUUID: "uuid-1",
		PoolsDefault: couchbase.TestPoolsDefaultData{
			ClusterName:   "test-cluster",
			StorageTotals: couchbase.TestStorageTotals{},
			Nodes: []struct {
				Version string `json:"version"`
			}{
				{
					Version: "7.0.0-0000-enterprise",
				},
			},
		},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "N0",
				Services:          []string{"kv"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports: map[string]uint16{
					"httpsMgmt": 9000,
				},
			},
		},
		Buckets:          []couchbase.BucketsEndpointData{},
		NodesReturnCode:  http.StatusOK,
		BucketReturnCode: http.StatusOK,
	}

	testHandler.Start(t, true)
	defer testHandler.Close()

	noSchemaHost := testHandler.URL()[len("https://"):]
	_, port, err := net.SplitHostPort(noSchemaHost)
	if err != nil {
		t.Fatalf("could not split hosts port: %v", err)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("Invalid port: %v", err)
	}

	testHandler.Nodes = []couchbase.TestNode{
		{
			NodeUUID:          "N0",
			Hostname:          noSchemaHost,
			Services:          []string{"kv"},
			Version:           "7.0.0-0000-enterprise",
			Status:            "healthy",
			ClusterMembership: "active",
			Ports: map[string]uint16{
				"httpsMgmt": uint16(portNum),
			},
		},
	}

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "cluster-0",
		User:     "user",
		Password: "password",
		BucketsSummary: values.BucketsSummary{
			{
				Name: "B0",
			},
		},
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "N0",
				Version:           "7.0.0-0000-enterprise",
				Host:              testHandler.URL(),
				Status:            "warmup",
				ClusterMembership: "active",
				Services:          []string{"kv", "backup"},
			},
		},
		ClusterInfo: &values.ClusterInfo{},
	}

	err = store.AddCluster(cluster)
	if err != nil {
		t.Fatalf("could not add test cluster: %v", err)
	}

	beforeHeartBeat := time.Now()

	monitor := NewMonitor(store, 1)
	monitor.Start(200 * time.Millisecond)
	time.Sleep(1 * time.Second)
	monitor.Stop()

	outCluster, err := store.GetCluster("uuid-0", true)
	if err != nil {
		t.Fatalf("Unexpected error retrieving the cluster: %v", err)
	}

	if outCluster.LastUpdate.Before(beforeHeartBeat) {
		t.Fatalf("Expected the last update to be after %v got %v", beforeHeartBeat, outCluster.LastUpdate)
	}

	//  things we do not want to compare
	outCluster.LastUpdate = time.Time{}
	outCluster.ClusterInfo = nil
	outCluster.CaCert = nil

	cluster.ClusterInfo = nil
	cluster.CaCert = nil
	cluster.LastUpdate = time.Time{}
	cluster.HeartBeatIssue = values.UUIDMismatchHeartIssue

	if !reflect.DeepEqual(cluster, outCluster) {
		t.Fatalf("Value does not match expected:\n%+v\n%+v", cluster, outCluster)
	}
}
