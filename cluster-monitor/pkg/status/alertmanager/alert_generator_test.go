// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package alertmanager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager/types"
	storagemocks "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// checkNumAlerts creates a function to pass into mock.Run,
// which checks that the right number of alerts have been passed.
func checkNumAlerts(t *testing.T, n int, msg string) func(arguments mock.Arguments) {
	return func(arguments mock.Arguments) {
		alerts := arguments.Get(1).([]types.PostableAlert)
		require.Equal(t, n, len(alerts), msg)
	}
}

func TestAlertGeneratorCreation(t *testing.T) {
	store := new(storagemocks.Store)
	ag := NewAlertGenerator(store, time.Minute, []string{"test1", "test2"}, nil)

	require.Len(t, ag.alertmanagers, 2)
}

// TestAlertGeneratorLabelsAnnotations verifies that the alerts we send have the expected metadata.
func TestAlertGeneratorLabelsAnnotations(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	ag := NewAlertGenerator(store, time.Minute, nil, nil)
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, []types.PostableAlert{
		{
			Labels: map[string]string{
				"cluster_name":      "Cluster 0",
				"health_check_id":   "CB90002",
				"health_check_name": "singleOrTwoNodeCluster",
				"job":               "couchbase_cluster_monitor",
				"kind":              "cluster",
				"severity":          "info",
			},
			Annotations: map[string]string{
				//nolint:lll
				"description": "Checks that production clusters have at least three nodes as clusters with fewer nodes cannot use some features.",
				"remediation": "TEST",
				"summary":     "Single or Two Node Cluster",
			},
			StartsAt:     &testTime,
			EndsAt:       nil,
			GeneratorURL: "",
		},
	}).Return(nil).Once()

	err := ag.update(context.Background())
	require.NoError(t, err)

	client.AssertExpectations(t)
}

// TestAlertGeneratorLifecycle verifies that we correctly transition an alert through the lifecycle: firing -> inactive
// -> gone
func TestAlertGeneratorLifecycle(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	clock := new(mocks.Clock)
	ag := NewAlertGenerator(store, time.Minute, nil, nil)
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}
	ag.clock = clock

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "initial")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))

	// Now remove the checker from the list, and check it still gets sent but in an inactive state
	testTime = testTime.Add(1 * time.Minute)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "inactive")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
	require.NotNilf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].EndsAt, "alert EndsAt nil")

	// Now go 10 minutes forward in time and check it's still sent
	testTime = testTime.Add(10 * time.Minute)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1,
		"still inactive")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
	require.NotNilf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].EndsAt, "alert EndsAt nil")

	// Now go another 5 minutes forward in time (plus a little margin) and check it's no longer sent
	testTime = testTime.Add(6 * time.Minute)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 0, "expired")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
}

// TestAlertGeneratorLifecycle verifies that we correctly handle a checker resolving, then un-resolving itself
func TestAlertGeneratorRecreate(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	clock := new(mocks.Clock)
	ag := NewAlertGenerator(store, time.Minute, nil, nil)
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}
	ag.clock = clock

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "initial")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))

	// Now reset the status to Good, and check it still gets sent but in an inactive state
	testTime = testTime.Add(1 * time.Minute)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:   values.CheckSingleOrTwoNodeCluster,
				Status: values.GoodCheckerStatus,
				Time:   testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "good")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
	require.NotNilf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].EndsAt, "alert EndsAt nil")

	// Now reset the status back to Info, and check we send a new alert
	// ("Any alerts in future evaluations with the same labels as an inactive alert MUST be considered as a new alert
	// and MUST follow the pending and firing state conditions as stated above")
	testTime = testTime.Add(1 * time.Minute)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.WarnCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 2, "reset")).Return(nil)

	clock.On("Now").Once().Return(testTime)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
}

// TestAlertGeneratorChangingAnnotations verifies that changing annotations (e.g. remediation) is handled appropriately
// (kept in the same alert, not a new one)
func TestAlertGeneratorChangingAnnotations(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	ag := NewAlertGenerator(store, time.Minute, nil, nil)
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "initial")).Return(nil)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "CHANGED",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1,
		"after change")).Return(nil)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)
	require.Equalf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].
		Annotations["remediation"], "CHANGED", "changed remediation")
}

// TestAlertGeneratorChangingLabels verifies that a checker changing its status results in a new alert,
// rather than a change to the existing one (which is forbidden since labels are immutable)
func TestAlertGeneratorChangingLabels(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	ag := NewAlertGenerator(store, time.Minute, nil, nil)
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 1, "initial")).Return(nil)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)

	// Now move the checker result to a warning, and check we fire two alerts of which one is inactive
	store.On("GetCheckerResult", mock.Anything).Once().Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.WarnCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	client.On("PostAlerts", mock.Anything, mock.Anything).Once().Run(checkNumAlerts(t, 2,
		"after change")).Return(nil)

	require.NoError(t, ag.update(context.Background()))
	client.AssertExpectations(t)

	require.Equalf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[1].
		Labels["severity"], "info", "initial severity")
	require.NotNilf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[1].EndsAt, "first alert inactive")

	require.Equalf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].
		Labels["severity"], "warning", "new severity")
	require.Nilf(t, client.Calls[1].Arguments[1].([]types.PostableAlert)[0].EndsAt, "second alert still firing")
}

func TestAlertGeneratorBaseLabelsAnnotations(t *testing.T) {
	store := new(storagemocks.Store)
	client := new(mocks.AlertmanagerClientIFace)
	ag := NewAlertGenerator(store, time.Minute, nil, map[string]string{"namespace": "database"})
	ag.alertmanagers = map[string]alertmanagerClientIFace{"test": client}

	testTime := time.Now()

	store.On("GetCheckerResult", mock.Anything).Return([]*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:        values.CheckSingleOrTwoNodeCluster,
				Status:      values.InfoCheckerStatus,
				Remediation: "TEST",
				Time:        testTime,
			},
			Cluster: "C0",
		},
	}, nil)

	store.On("GetCluster", "C0", false).Return(&values.CouchbaseCluster{
		Name: "Cluster 0",
	}, nil)

	client.On("PostAlerts", mock.Anything, []types.PostableAlert{
		{
			Labels: map[string]string{
				"cluster_name":      "Cluster 0",
				"health_check_id":   "CB90002",
				"health_check_name": "singleOrTwoNodeCluster",
				"job":               "couchbase_cluster_monitor",
				"kind":              "cluster",
				"severity":          "info",
				"namespace":         "database",
			},
			Annotations: map[string]string{
				//nolint:lll
				"description": "Checks that production clusters have at least three nodes as clusters with fewer nodes cannot use some features.",
				"remediation": "TEST",
				"summary":     "Single or Two Node Cluster",
			},
			StartsAt:     &testTime,
			EndsAt:       nil,
			GeneratorURL: "",
		},
	}).Return(nil).Once()

	err := ag.update(context.Background())
	require.NoError(t, err)

	client.AssertExpectations(t)
}
