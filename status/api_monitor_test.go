package status

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/status/progress"
	"github.com/couchbaselabs/cbmultimanager/storage/sqlite"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeChecker is used to easily test that the checkers are run by the monitor.
type fakeChecker struct {
	runs int

	// for checker function
	returnValue []*values.WrappedCheckerResult
	err         error
}

func (f *fakeChecker) checkerFN(*values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	f.runs++
	return f.returnValue, f.err
}

func TestAPIMonitorWorking(t *testing.T) {
	// setup test store
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store"), "key")
	if err != nil {
		t.Fatalf("Could not create test store: %v", err)
	}

	// setup test data
	err = store.AddCluster(&values.CouchbaseCluster{
		UUID:         "UUID-0",
		Name:         "C0",
		User:         "user",
		Password:     "pass",
		NodesSummary: values.NodesSummary{{}},
		LastUpdate:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Unexpected error adding test class: %v", err)
	}

	// setup test checkers
	testChecker := &fakeChecker{
		returnValue: []*values.WrappedCheckerResult{
			{
				Cluster: "UUID-0",
				Result: &values.CheckerResult{
					Name:   "checkerA",
					Status: values.GoodCheckerStatus,
					Time:   time.Now().UTC(),
				},
			},
			{
				Error: fmt.Errorf("some error here"),
			},
		},
	}

	testCheckers := map[string]values.CheckerFn{"testChecker": testChecker.checkerFN}

	// create monitor
	monitor := &apiMonitor{
		store:           store,
		workerWg:        &sync.WaitGroup{},
		numWorkers:      1,
		streamSize:      5,
		trigger:         make(chan struct{}, 5),
		checkers:        testCheckers,
		progressMonitor: progress.NewMonitor(),
	}

	// start monitor and wait a bit then check that the checker run as expected
	monitor.start(200 * time.Millisecond)
	time.Sleep(time.Second)
	monitor.stop()

	if testChecker.runs < 2 || testChecker.runs > 6 {
		t.Fatalf("Expected the checks to run between from 2 to 6 but run %d", testChecker.runs)
	}

	results, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &testChecker.returnValue[0].Cluster})
	if err != nil {
		t.Fatalf("Unexpected error getting the checker results: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result got %d", len(results))
	}

	wrappedResultMustMatch(testChecker.returnValue[0], results[0], false, true, t)
}

func TestAPIMonitorTrigger(t *testing.T) {
	// setup test store
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store"), "key")
	if err != nil {
		t.Fatalf("Could not create test store: %v", err)
	}

	// setup test data
	err = store.AddCluster(&values.CouchbaseCluster{
		UUID:         "UUID-0",
		Name:         "C0",
		User:         "user",
		Password:     "pass",
		NodesSummary: values.NodesSummary{{}},
		LastUpdate:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Unexpected error adding test class: %v", err)
	}

	// setup test checkers
	testChecker := &fakeChecker{
		returnValue: []*values.WrappedCheckerResult{
			{
				Cluster: "UUID-0",
				Result: &values.CheckerResult{
					Name:   "checkerA",
					Status: values.GoodCheckerStatus,
					Time:   time.Now().UTC(),
				},
			},
			{
				Error: fmt.Errorf("some error here"),
			},
		},
	}

	testCheckers := map[string]values.CheckerFn{"testChecker": testChecker.checkerFN}

	// create monitor
	monitor := &apiMonitor{
		store:           store,
		workerWg:        &sync.WaitGroup{},
		numWorkers:      1,
		streamSize:      5,
		trigger:         make(chan struct{}, 5),
		checkers:        testCheckers,
		progressMonitor: progress.NewMonitor(),
	}

	// start monitor and wait a bit then check that the checker run as expected
	monitor.start(10 * time.Minute)

	// trigger twice in a row
	_ = monitor.triggerCheck()
	_ = monitor.triggerCheck()

	// wait for a while and then stop
	time.Sleep(time.Second)
	monitor.stop()

	if testChecker.runs != 3 {
		t.Fatalf("Expected the checks to run 3 times %d", testChecker.runs)
	}

	results, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &testChecker.returnValue[0].Cluster})
	if err != nil {
		t.Fatalf("Unexpected error getting the checker results: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result got %d", len(results))
	}

	wrappedResultMustMatch(testChecker.returnValue[0], results[0], false, true, t)
}

func TestTriggerFor(t *testing.T) {
	// setup test store
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store"), "key")
	require.NoError(t, err)

	// setup test data
	cluster := &values.CouchbaseCluster{
		UUID:         "UUID-0",
		Name:         "C0",
		User:         "user",
		Password:     "pass",
		NodesSummary: values.NodesSummary{{}},
		LastUpdate:   time.Now().UTC(),
	}
	require.NoError(t, store.AddCluster(cluster))

	// setup test checkers
	testChecker := &fakeChecker{
		returnValue: []*values.WrappedCheckerResult{
			{
				Cluster: "UUID-0",
				Result: &values.CheckerResult{
					Name:   "checkerA",
					Status: values.GoodCheckerStatus,
					Time:   time.Now().UTC(),
				},
			},
			{
				Error: assert.AnError,
			},
		},
	}

	testCheckers := map[string]values.CheckerFn{"testChecker": testChecker.checkerFN}

	// create monitor
	monitor := &apiMonitor{
		store:           store,
		workerWg:        &sync.WaitGroup{},
		numWorkers:      1,
		streamSize:      5,
		trigger:         make(chan struct{}, 5),
		checkers:        testCheckers,
		progressMonitor: progress.NewMonitor(),
	}

	// start monitor and wait a bit then check that the checker run as expected
	monitor.start(10 * time.Minute)

	// trigger twice in a row
	require.NoError(t, monitor.triggerFor(cluster))
	require.NoError(t, monitor.triggerFor(cluster))

	// wait for a while and then stop
	time.Sleep(time.Second)
	monitor.stop()

	require.Equal(t, 3, testChecker.runs, "unexpected number of runs")

	results, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &testChecker.returnValue[0].Cluster})
	require.NoError(t, err)
	require.Len(t, results, 1)

	wrappedResultMustMatch(testChecker.returnValue[0], results[0], false, true, t)
}
