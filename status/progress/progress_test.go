package progress

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/values"
	"github.com/stretchr/testify/require"
)

func getTestProgressMonitor() *Monitor {
	progress := NewMonitor()
	progress.StartChecking([]*values.CouchbaseCluster{
		{
			UUID: "0",
		},
		{
			UUID: "1",
		},
		{
			UUID: "2",
		},
		{
			UUID: "3",
		},
	})

	return progress
}

func TestProgressMonitorStart(t *testing.T) {
	progress := getTestProgressMonitor()
	expectedProgress := &Monitor{
		inProgress: true,
		lastRun:    progress.lastRun,
		clusterProgress: values.ClusterProgressMap{
			"0": &values.ClusterProgress{Status: values.Waiting},
			"1": &values.ClusterProgress{Status: values.Waiting},
			"2": &values.ClusterProgress{Status: values.Waiting},
			"3": &values.ClusterProgress{Status: values.Waiting},
		},
	}

	require.Equal(t, expectedProgress, progress)

	progress.FinishChecking()
	require.False(t, progress.inProgress)
}

func TestProgressMonitorClusterChecking(t *testing.T) {
	progress := getTestProgressMonitor()

	expectedProgress := &Monitor{
		inProgress: true,
		lastRun:    progress.lastRun,
		clusterProgress: values.ClusterProgressMap{
			"0": &values.ClusterProgress{Status: values.InProgress, TotalCheckers: 10},
			"1": &values.ClusterProgress{Status: values.Waiting},
			"2": &values.ClusterProgress{Status: values.Waiting},
			"3": &values.ClusterProgress{Status: values.Waiting},
		},
	}

	t.Run("start", func(t *testing.T) {
		progress.ClusterRunStart("0", 10)
		expectedProgress.clusterProgress["0"].Start = progress.clusterProgress["0"].Start
		require.Equal(t, expectedProgress, progress)
		require.NotZero(t, expectedProgress.clusterProgress["0"].Start)
	})

	t.Run("checker-success", func(t *testing.T) {
		require.NoError(t, progress.CheckerDone("0", false))
		expectedProgress.clusterProgress["0"].Done++
		require.Equal(t, expectedProgress, progress)
	})

	t.Run("checker-failed", func(t *testing.T) {
		require.NoError(t, progress.CheckerDone("0", true))
		expectedProgress.clusterProgress["0"].Failed++
		require.Equal(t, expectedProgress, progress)
	})

	t.Run("checker-end", func(t *testing.T) {
		require.NoError(t, progress.ClusterRunEnd("0"))
		expectedProgress.clusterProgress["0"].Status = values.Done
		expectedProgress.clusterProgress["0"].End = progress.clusterProgress["0"].End
		require.NotZero(t, expectedProgress.clusterProgress["0"].Start)
	})
}

func TestProgressMonitorGetProgressFor(t *testing.T) {
	progress := &Monitor{
		clusterProgress: values.ClusterProgressMap{
			"0": &values.ClusterProgress{Status: values.Done, Done: 7, Failed: 1, TotalCheckers: 8},
			"1": &values.ClusterProgress{Status: values.InProgress, Done: 2, TotalCheckers: 8},
			"2": &values.ClusterProgress{Status: values.Waiting, TotalCheckers: 7},
		},
	}

	t.Run("exist", func(t *testing.T) {
		for key, val := range progress.clusterProgress {
			t.Run(string(val.Status), func(t *testing.T) {
				out, err := progress.GetProgressFor(key)
				require.NoError(t, err)
				require.Equal(t, val, out)
				require.NotSame(t, val, out)
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		_, err := progress.GetProgressFor("not-here")
		var clusterNotFoundErr *ClusterNotFoundError
		require.ErrorAs(t, err, &clusterNotFoundErr)
	})
}
