// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func TestAddDismissal(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("no-id", func(t *testing.T) {
		if err := db.AddDismissal(values.Dismissal{}); err == nil {
			t.Fatalf("Should not be able to add a dismissal without ID")
		}
	})

	t.Run("all-fields", func(t *testing.T) {
		dismissal := values.Dismissal{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "A",
			BucketName:  "B",
			LogFile:     "C",
			NodeUUID:    "D",
			ID:          "id",
			CheckerName: "A",
		}

		if err := db.AddDismissal(dismissal); err != nil {
			t.Fatalf("Unexpected error dismissing: %v", err)
		}

		out, err := db.GetDismissals(values.DismissalSearchSpace{ID: &dismissal.ID})
		if err != nil {
			t.Fatalf("Unexpected error getting dismissal: %v", err)
		}

		if len(out) != 1 {
			t.Fatalf("Expected 1 dismissal to be returned got %d", len(out))
		}

		if !reflect.DeepEqual(out[0], &dismissal) {
			t.Fatalf("Expected %+v got %+v", &dismissal, out[0])
		}
	})
}

type dismissalTestCase struct {
	name     string
	expected []*values.Dismissal
	search   values.DismissalSearchSpace
}

func TestGetDismissals(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.Dismissal{
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N0",
			ID:          "0",
			CheckerName: "CH1",
		},
		{
			Forever:     true,
			Level:       values.ClusterDismissLevel,
			ClusterUUID: "C0",
			ID:          "1",
			CheckerName: "CH2",
		},
		{
			Until:       time.Now().Add(1 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "3",
			CheckerName: "CH2",
		},
		{
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ID:          "4",
			ClusterUUID: "C1",
			BucketName:  "B0",
			CheckerName: "CH3",
		},
	}

	for _, dismissal := range dataSet {
		if err := db.AddDismissal(*dismissal); err != nil {
			t.Fatalf("Unexpected error when adding dismissal: %v", err)
		}
	}

	cases := []dismissalTestCase{
		{
			name:     "get-all",
			expected: dataSet,
			search:   values.DismissalSearchSpace{},
		},
		{
			name:     "no-matches",
			expected: []*values.Dismissal{},
			search: values.DismissalSearchSpace{
				Level:       pointerLevel(values.BucketDismissLevel),
				ClusterUUID: &dataSet[0].ClusterUUID,
			},
		},
		{
			name:     "all-level",
			expected: dataSet[2:3],
			search: values.DismissalSearchSpace{
				Level: pointerLevel(values.AllDismissLevel),
			},
		},
		{
			name:     "cluster-dismissals",
			expected: dataSet[0:2],
			search: values.DismissalSearchSpace{
				ClusterUUID: &dataSet[0].ClusterUUID,
			},
		},
		{
			name:     "by-checker",
			expected: dataSet[1:3],
			search: values.DismissalSearchSpace{
				CheckerName: &dataSet[2].CheckerName,
			},
		},
		{
			name:     "by-cluster-and-bucket",
			expected: dataSet[3:4],
			search: values.DismissalSearchSpace{
				ClusterUUID: &dataSet[3].ClusterUUID,
				BucketName:  &dataSet[3].BucketName,
			},
		},
		{
			name:     "by-cluster-and-node",
			expected: dataSet[0:1],
			search: values.DismissalSearchSpace{
				ClusterUUID: &dataSet[0].ClusterUUID,
				NodeUUID:    &dataSet[0].NodeUUID,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := db.GetDismissals(tc.search)
			if err != nil {
				t.Fatalf("Unexpected error whilst getting dismissals: %v", err)
			}

			if !reflect.DeepEqual(out, tc.expected) {
				t.Fatalf("Values do not match.\n%+v\n%+v", tc.expected, out)
			}
		})
	}
}

func TestDeleteDismissals(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.Dismissal{
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N0",
			ID:          "0",
			CheckerName: "CH1",
		},
		{
			Forever:     true,
			Level:       values.ClusterDismissLevel,
			ClusterUUID: "C0",
			ID:          "1",
			CheckerName: "CH2",
		},
		{
			Until:       time.Now().Add(1 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "3",
			CheckerName: "CH2",
		},
		{
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ID:          "4",
			ClusterUUID: "C1",
			BucketName:  "B0",
			CheckerName: "CH3",
		},
	}

	for _, dismissal := range dataSet {
		if err := db.AddDismissal(*dismissal); err != nil {
			t.Fatalf("Unexpected error when adding dismissal: %v", err)
		}
	}

	cases := []dismissalTestCase{
		{
			name:     "no-match",
			search:   values.DismissalSearchSpace{CheckerName: stringPointer("CH4")},
			expected: dataSet,
		},
		{
			name:     "by-checker-name",
			search:   values.DismissalSearchSpace{CheckerName: stringPointer("CH3")},
			expected: dataSet[0:3],
		},
		{
			name:     "by-cluster",
			search:   values.DismissalSearchSpace{ClusterUUID: stringPointer("C0")},
			expected: dataSet[2:3],
		},
		{
			name:     "by-id",
			search:   values.DismissalSearchSpace{ID: stringPointer("3")},
			expected: []*values.Dismissal{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.DeleteDismissals(tc.search); err != nil {
				t.Fatalf("Unexpected error deleting dismissal: %v", err)
			}

			out, err := db.GetDismissals(values.DismissalSearchSpace{})
			if err != nil {
				t.Fatalf("Unexpected error getting dismissals: %v", err)
			}

			if !reflect.DeepEqual(out, tc.expected) {
				t.Fatalf("Values do not match.\n%+v\n%+v", tc.expected, out)
			}
		})
	}
}

func TestDeleteExpiredDismissals(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.Dismissal{
		{
			Until:       time.Now().Add(-time.Minute).UTC(),
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N0",
			ID:          "0",
			CheckerName: "CH1",
		},
		{
			Until:       time.Now().Add(-time.Hour).UTC(),
			Level:       values.ClusterDismissLevel,
			ClusterUUID: "C0",
			ID:          "1",
			CheckerName: "CH2",
		},
		{
			Until:       time.Now().Add(1 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "3",
			CheckerName: "CH2",
		},
		{
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ID:          "4",
			ClusterUUID: "C1",
			BucketName:  "B0",
			CheckerName: "CH3",
		},
	}

	for _, dismissal := range dataSet {
		if err := db.AddDismissal(*dismissal); err != nil {
			t.Fatalf("Unexpected error when adding dismissal: %v", err)
		}
	}

	n, err := db.DeleteExpiredDismissals()
	if err != nil {
		t.Fatalf("Unexpected error deleting expired dismissals")
	}

	if n != 2 {
		t.Fatalf("Expected 2 items to be deleted instead %d were deleted", n)
	}

	out, err := db.GetDismissals(values.DismissalSearchSpace{})
	if err != nil {
		t.Fatalf("Unexpected error getting dismissals")
	}

	if !reflect.DeepEqual(out, dataSet[2:4]) {
		t.Fatalf("Values do not match.\n%+v\n%+v", dataSet[2:4], out)
	}
}

func TestDB_DeleteDismissalForUnknownBuckets(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.Dismissal{
		{
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ID:          "0",
			ClusterUUID: "C1",
			BucketName:  "B0",
			CheckerName: "CH3",
		},
		{
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			BucketName:  "B0",
			ID:          "1",
			CheckerName: "CH1",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			BucketName:  "B1",
			ID:          "2",
			CheckerName: "CH2",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			BucketName:  "B2",
			ID:          "3",
			CheckerName: "CH2",
		},
		{
			Until:       time.Now().Add(1 * time.Hour).UTC(),
			Level:       values.FileDismissLevel,
			ClusterUUID: "C0",
			BucketName:  "B3",
			ID:          "4",
			CheckerName: "CH2",
		},
	}

	for _, dismissal := range dataSet {
		if err := db.AddDismissal(*dismissal); err != nil {
			t.Fatalf("Unexpected error when adding dismissal: %v", err)
		}
	}

	n, err := db.DeleteDismissalForUnknownBuckets([]string{"B0", "B1"}, "C0")
	if err != nil {
		t.Fatalf("Unexpected error deleting dismissals for unknown nodes")
	}

	if n != 2 {
		t.Fatalf("Expected 2 elemented to be delted instead %d were deleted", n)
	}

	out, err := db.GetDismissals(values.DismissalSearchSpace{})
	if err != nil {
		t.Fatalf("Unexpected error getting dismissals")
	}

	if !reflect.DeepEqual(out, dataSet[0:3]) {
		t.Fatalf("Values do not match.\n%+v\n%+v", dataSet[0:3], out)
	}
}

func TestDeleteDismissalForUnknownNodes(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.Dismissal{
		{
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ID:          "0",
			ClusterUUID: "C1",
			BucketName:  "B0",
			CheckerName: "CH3",
		},
		{
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N0",
			ID:          "1",
			CheckerName: "CH1",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N1",
			ID:          "2",
			CheckerName: "CH2",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N2",
			ID:          "3",
			CheckerName: "CH2",
		},
		{
			Until:       time.Now().Add(1 * time.Hour).UTC(),
			Level:       values.FileDismissLevel,
			ClusterUUID: "C0",
			NodeUUID:    "N2",
			ID:          "4",
			LogFile:     "L0",
			CheckerName: "CH2",
		},
	}

	for _, dismissal := range dataSet {
		if err := db.AddDismissal(*dismissal); err != nil {
			t.Fatalf("Unexpected error when adding dismissal: %v", err)
		}
	}

	n, err := db.DeleteDismissalForUnknownNodes([]string{"N0", "N1"}, "C0")
	if err != nil {
		t.Fatalf("Unexpected error deleting dismissals for unknown nodes")
	}

	if n != 2 {
		t.Fatalf("Expected 2 elemented to be delted instead %d were deleted", n)
	}

	out, err := db.GetDismissals(values.DismissalSearchSpace{})
	if err != nil {
		t.Fatalf("Unexpected error getting dismissals")
	}

	if !reflect.DeepEqual(out, dataSet[0:3]) {
		t.Fatalf("Values do not match.\n%+v\n%+v", dataSet[0:3], out)
	}
}

func pointerLevel(level values.DismissLevel) *values.DismissLevel {
	return &level
}

func stringPointer(str string) *string {
	return &str
}
