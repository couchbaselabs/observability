package status

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/status/progress"
	"github.com/couchbaselabs/cbmultimanager/storage/sqlite"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/require"
)

type testChecker struct {
	results []*values.WrappedCheckerResult
	err     error

	uuids []string
}

func (t *testChecker) check(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	if t.uuids == nil {
		t.uuids = make([]string, 0)
	}

	t.uuids = append(t.uuids, cluster.UUID)
	if len(t.results) != 0 {
		for _, result := range t.results {
			result.Cluster = cluster.UUID
		}
	}

	return t.results, t.err
}

// TestAPIWorker will spawn a worker and feed it some test checkers and clusters. After all the checks are run it will
// check the store and ensure the expected results are stored.
func TestAPIWorker(t *testing.T) {
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "storage.sqlite"), "key")
	require.NoError(t, err)

	stream := make(chan *values.CouchbaseCluster, 2)
	wg := &sync.WaitGroup{}

	checkers := []*testChecker{
		{
			results: nil,
			err:     values.ErrNotFound,
		},
		{
			results: []*values.WrappedCheckerResult{
				{
					Error: values.ErrNotFound,
				},
			},
		},
		{
			results: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "A",
						Status: values.GoodCheckerStatus,
						Time:   time.Now().UTC(),
					},
					Node: "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:        "A",
						Remediation: "Do something",
						Value:       []byte(`"The boat is sinking"`),
						Status:      values.AlertCheckerStatus,
						Time:        time.Now().UTC(),
					},
					Node: "N1",
				},
			},
		},
	}

	checkerFns := map[string]values.CheckerFn{
		"error":        checkers[0].check,
		"inside_error": checkers[1].check,
		"results":      checkers[2].check,
	}

	stream <- &values.CouchbaseCluster{UUID: "C1"}
	stream <- &values.CouchbaseCluster{UUID: "C2"}
	close(stream)

	wg.Add(1)
	go (&apiWorker{
		store:           store,
		progressMonitor: progress.NewMonitor(),
		checkers:        checkerFns,
		ctx:             context.Background(),
		stream:          stream,
		wg:              wg,
	}).start()

	wg.Wait()

	results, err := store.GetCheckerResult(values.CheckerSearch{})
	require.NoError(t, err)
	require.Len(t, results, 4)

	for _, clusterUUID := range []string{"C1", "C2"} {
		clusterResults, err := store.GetCheckerResult(values.CheckerSearch{Cluster: stringPointer(clusterUUID)})
		require.NoError(t, err)
		require.Len(t, clusterResults, 2)

		for i, result := range clusterResults {
			// replace the cluster uuid
			checkers[2].results[i].Cluster = clusterUUID
			require.Equal(t, checkers[2].results[i], result)
		}
	}
}

func stringPointer(s string) *string { return &s }
