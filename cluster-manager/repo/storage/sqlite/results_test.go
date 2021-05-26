package sqlite

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

type checkerTestCase struct {
	name        string
	expected    []*values.WrappedCheckerResult
	searchSpace values.CheckerSearch
}

func TestGetCheckerResult(t *testing.T) {
	db, _ := createEmptyDB(t)

	dataSet := []*values.WrappedCheckerResult{
		{
			Cluster: "C0",
			Result: &values.CheckerResult{
				Name:        "A",
				Remediation: "A",
				Value:       []byte(`"some-value-1"`),
				Status:      values.AlertCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "B",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "C",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b1",
			Result: &values.CheckerResult{
				Name:        "C",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.InfoCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n0",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-2"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n2",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			LogFile: "l0",
			Result: &values.CheckerResult{
				Name:   "E",
				Value:  []byte(`"some-value-3"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
	}

	// insert the initial data set
	for _, result := range dataSet {
		if err := db.SetCheckerResult(result); err != nil {
			t.Fatalf("Unexpected error inserting test data set: %v", err)
		}
	}

	cases := []checkerTestCase{
		{
			name:        "no-matches",
			searchSpace: values.CheckerSearch{Cluster: stringPointer("c5")},
		},
		{
			name:        "search-by-cluster",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[0].Cluster},
			expected:    dataSet[0:6],
		},
		{
			name:        "search-by-cluster-and-bucket",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[0].Cluster, Bucket: &dataSet[1].Bucket},
			expected:    dataSet[1:3],
		},
		{
			name:        "search-by-cluster-and-node",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[6].Cluster, Node: &dataSet[6].Node},
			expected:    dataSet[6:8],
		},
		{
			name: "search-by-cluster-node-and-log",
			searchSpace: values.CheckerSearch{
				Cluster: &dataSet[7].Cluster,
				Node:    &dataSet[7].Node,
				LogFile: &dataSet[7].LogFile,
			},
			expected: dataSet[7:8],
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := db.GetCheckerResult(tc.searchSpace)
			if err != nil {
				t.Fatalf("Unexpected error getting results: %v", err)
			}

			if len(out) != len(tc.expected) {
				t.Fatalf("Expected %d got %d results", len(tc.expected), len(out))
			}

			for i, res := range out {
				resultsMustMatch(tc.expected[i], res, t)
			}
		})
	}
}

func TestSetCheckerResult(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("with-errors", func(t *testing.T) {
		err := db.SetCheckerResult(&values.WrappedCheckerResult{
			Result:  &values.CheckerResult{},
			Error:   fmt.Errorf("test-error"),
			Cluster: "c0",
		})
		if err == nil {
			t.Fatal("Should not be able to store results with errors")
		}
	})

	t.Run("without-result", func(t *testing.T) {
		err := db.SetCheckerResult(&values.WrappedCheckerResult{Cluster: "c0"})
		if err == nil {
			t.Fatal("Should not be able to store value without results")
		}
	})

	t.Run("valid-cluster-level", func(t *testing.T) {
		result := &values.WrappedCheckerResult{
			Cluster: "C0",
			Result: &values.CheckerResult{
				Name:        "A",
				Remediation: "A",
				Value:       []byte(`"some-value"`),
				Status:      values.GoodCheckerStatus,
				Time:        time.Now(),
			},
		}

		if err := db.SetCheckerResult(result); err != nil {
			t.Fatalf("unexpected error adding checker: %v", err)
		}

		out, err := db.GetCheckerResult(values.CheckerSearch{Cluster: &result.Cluster})
		if err != nil {
			t.Fatalf("Unexpected error getting results: %v", err)
		}

		if len(out) != 1 {
			t.Fatalf("Expected 1 element got %d", len(out))
		}

		resultsMustMatch(result, out[0], t)
	})

	t.Run("valid-node-level", func(t *testing.T) {
		results := []*values.WrappedCheckerResult{
			{
				Cluster: "C0",
				Node:    "n0",
				Result: &values.CheckerResult{
					Name:        "A",
					Remediation: "A",
					Value:       []byte(`"some-value-1"`),
					Status:      values.GoodCheckerStatus,
					Time:        time.Now(),
				},
			},
			{
				Cluster: "C0",
				Node:    "n0",
				Result: &values.CheckerResult{
					Name:        "B",
					Remediation: "A",
					Value:       []byte(`"some-value-2"`),
					Status:      values.AlertCheckerStatus,
					Time:        time.Now(),
				},
			},
		}

		for _, res := range results {
			if err := db.SetCheckerResult(res); err != nil {
				t.Fatalf("unexpected error adding checker: %v", err)
			}
		}

		out, err := db.GetCheckerResult(values.CheckerSearch{Cluster: &results[0].Cluster, Node: &results[0].Node})
		if err != nil {
			t.Fatalf("Unexpected error getting results: %v", err)
		}

		if len(out) != 2 {
			t.Fatalf("Expected 2 elements got %d", len(out))
		}

		for i, outRes := range out {
			resultsMustMatch(results[i], outRes, t)
		}
	})

	t.Run("valid-bucket-level", func(t *testing.T) {
		results := []*values.WrappedCheckerResult{
			{
				Cluster: "C0",
				Bucket:  "b0",
				Result: &values.CheckerResult{
					Name:        "A",
					Remediation: "A",
					Value:       []byte(`"some-value-1"`),
					Status:      values.GoodCheckerStatus,
					Time:        time.Now(),
				},
			},
			{
				Cluster: "C0",
				Bucket:  "b0",
				Result: &values.CheckerResult{
					Name:        "B",
					Remediation: "A",
					Value:       []byte(`"some-value-2"`),
					Status:      values.InfoCheckerStatus,
					Time:        time.Now(),
				},
			},
		}

		for _, res := range results {
			if err := db.SetCheckerResult(res); err != nil {
				t.Fatalf("unexpected error adding checker: %v", err)
			}
		}

		out, err := db.GetCheckerResult(values.CheckerSearch{Cluster: &results[0].Cluster, Bucket: &results[0].Bucket})
		if err != nil {
			t.Fatalf("Unexpected error getting results: %v", err)
		}

		if len(out) != 2 {
			t.Fatalf("Expected 2 elements got %d", len(out))
		}

		for i, outRes := range out {
			resultsMustMatch(results[i], outRes, t)
		}
	})
}

func TestDeleteCheckerResults(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.WrappedCheckerResult{
		{
			Cluster: "C0",
			Result: &values.CheckerResult{
				Name:        "A",
				Remediation: "A",
				Value:       []byte(`"some-value-1"`),
				Status:      values.AlertCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "B",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "C",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b1",
			Result: &values.CheckerResult{
				Name:        "C",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.InfoCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n0",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-2"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n2",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			LogFile: "l0",
			Result: &values.CheckerResult{
				Name:   "E",
				Value:  []byte(`"some-value-3"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
	}

	// insert the initial data set
	for _, result := range dataSet {
		if err := db.SetCheckerResult(result); err != nil {
			t.Fatalf("Unexpected error inserting test data set: %v", err)
		}
	}

	t.Run("no-search-space", func(t *testing.T) {
		err := db.DeleteCheckerResults(values.CheckerSearch{})
		if err == nil {
			t.Fatal("Expected at least 1 term to be required in the search space")
		}
	})

	cases := []checkerTestCase{
		{
			name:        "delete-by-cluster-that-does-not-exist",
			searchSpace: values.CheckerSearch{Cluster: stringPointer("fakeCluster")},
			expected:    dataSet,
		},
		{
			name: "delete-by-cluster-node-and-log-file",
			searchSpace: values.CheckerSearch{
				Cluster: &dataSet[7].Cluster,
				Node:    &dataSet[7].Node,
				LogFile: &dataSet[7].LogFile,
			},
			expected: dataSet[0:7],
		},
		{
			name:        "delete-by-cluster",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[6].Cluster},
			expected:    dataSet[0:6],
		},
		{
			name:        "delete-by-cluster-and-node",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[5].Cluster, Node: &dataSet[5].Node},
			expected:    dataSet[0:5],
		},
		{
			name:        "delete-by-cluster-and-bucket",
			searchSpace: values.CheckerSearch{Cluster: &dataSet[1].Cluster, Bucket: &dataSet[1].Bucket},
			expected:    []*values.WrappedCheckerResult{dataSet[0], dataSet[3], dataSet[4]},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.DeleteCheckerResults(tc.searchSpace); err != nil {
				t.Fatalf("Error deleting checker results: %v", err)
			}

			results, err := db.GetCheckerResult(values.CheckerSearch{})
			if err != nil {
				t.Fatalf("Unexpected error getting the results: %v", err)
			}

			if len(results) != len(tc.expected) {
				t.Fatalf("Expected %d got %d results", len(tc.expected), len(results))
			}

			for i, res := range results {
				resultsMustMatch(tc.expected[i], res, t)
			}
		})
	}
}

func TestDeleteWhereNodesDoNotMatch(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.WrappedCheckerResult{
		{
			Cluster: "C0",
			Result: &values.CheckerResult{
				Name:        "A",
				Remediation: "A",
				Value:       []byte(`"some-value-1"`),
				Status:      values.AlertCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "B",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n0",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-2"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n1",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n2",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			LogFile: "l0",
			Result: &values.CheckerResult{
				Name:   "E",
				Value:  []byte(`"some-value-3"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
	}

	// insert the initial data set
	for _, result := range dataSet {
		if err := db.SetCheckerResult(result); err != nil {
			t.Fatalf("Unexpected error inserting test data set: %v", err)
		}
	}

	t.Run("nil", func(t *testing.T) {
		if _, err := db.DeleteWhereNodesDoNotMatch(nil, "C0"); err == nil {
			t.Fatal("Expected an error when no nodes given instead got <nil>")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := db.DeleteWhereNodesDoNotMatch([]string{}, "C0"); err == nil {
			t.Fatal("Expected an error when no nodes given instead got <nil>")
		}
	})

	t.Run("nodes", func(t *testing.T) {
		deleted, err := db.DeleteWhereNodesDoNotMatch([]string{"n0", "n1"}, "C0")
		if err != nil {
			t.Fatalf("Unexpected error deleting results: %v", err)
		}

		if deleted != 1 {
			t.Fatalf("Expected 1 element to be deleted instead %d were deleted", deleted)
		}

		results, err := db.GetCheckerResult(values.CheckerSearch{})
		if err != nil {
			t.Fatalf("Unexpected error getting checker results")
		}

		expected := append(dataSet[0:4], dataSet[5])
		if len(results) != len(expected) {
			t.Fatalf("Expected %d items to remain instead got %d", len(expected), len(results))
		}

		for i, res := range results {
			resultsMustMatch(expected[i], res, t)
		}
	})
}

func TestDeleteWhereBucketsDoNotMatch(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	dataSet := []*values.WrappedCheckerResult{
		{
			Cluster: "C0",
			Result: &values.CheckerResult{
				Name:        "A",
				Remediation: "A",
				Value:       []byte(`"some-value-1"`),
				Status:      values.AlertCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b0",
			Result: &values.CheckerResult{
				Name:        "B",
				Remediation: "A",
				Value:       []byte(`"some-value-2"`),
				Status:      values.WarnCheckerStatus,
				Time:        time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b1",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-2"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Bucket:  "b2",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C0",
			Node:    "n0",
			Result: &values.CheckerResult{
				Name:   "D",
				Value:  []byte(`"some-value-3"`),
				Status: values.MissingCheckerStatus,
				Time:   time.Now(),
			},
		},
		{
			Cluster: "C1",
			Node:    "n3",
			LogFile: "l0",
			Result: &values.CheckerResult{
				Name:   "E",
				Value:  []byte(`"some-value-3"`),
				Status: values.GoodCheckerStatus,
				Time:   time.Now(),
			},
		},
	}

	// insert the initial data set
	for _, result := range dataSet {
		if err := db.SetCheckerResult(result); err != nil {
			t.Fatalf("Unexpected error inserting test data set: %v", err)
		}
	}

	t.Run("nil", func(t *testing.T) {
		if _, err := db.DeleteWhereBucketsDoNotMatch(nil, "C0"); err == nil {
			t.Fatal("Expected an error when no nodes given instead got <nil>")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := db.DeleteWhereBucketsDoNotMatch([]string{}, "C0"); err == nil {
			t.Fatal("Expected an error when no nodes given instead got <nil>")
		}
	})

	t.Run("nodes", func(t *testing.T) {
		deleted, err := db.DeleteWhereBucketsDoNotMatch([]string{"b0", "b1"}, "C0")
		if err != nil {
			t.Fatalf("Unexpected error deleting results: %v", err)
		}

		if deleted != 1 {
			t.Fatalf("Expected 1 element to be deleted instead %d were deleted", deleted)
		}

		results, err := db.GetCheckerResult(values.CheckerSearch{})
		if err != nil {
			t.Fatalf("Unexpected error getting checker results")
		}

		expected := []*values.WrappedCheckerResult{dataSet[0], dataSet[1], dataSet[2], dataSet[4], dataSet[5]}
		if len(results) != len(expected) {
			t.Fatalf("Expected %d items to remain instead got %d", len(expected), len(results))
		}

		for i, res := range results {
			resultsMustMatch(expected[i], res, t)
		}
	})
}

func resultsMustMatch(in, out *values.WrappedCheckerResult, t *testing.T) {
	if in.Cluster != out.Cluster || in.Bucket != out.Bucket || in.Node != out.Node || in.LogFile != out.LogFile {
		t.Fatalf("In and out wrapping do not match.\n%+v\n%v", in, out)
	}

	// time marshaling does not compare well so we just equate them
	out.Result.Time = in.Result.Time
	if !reflect.DeepEqual(in.Result, out.Result) {
		t.Fatalf("results do not match.\n%+v\n%+v", in.Result, out.Result)
	}
}
