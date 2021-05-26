package couchbase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/stretchr/testify/require"
)

func TestGetUILogs(t *testing.T) {
	var statusCode int
	uiLogs := UILogs{
		List: []UILogEntry{
			{
				Code:       10,
				Module:     "ns_server",
				ServerTime: "today",
				Text:       "hello",
				Type:       "info",
			},
			{
				Code:       1,
				Module:     "ns_server",
				ServerTime: "today",
				Text:       "hello",
				Type:       "info",
			},
		},
	}

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(UILogsEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, &uiLogs, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("404", func(t *testing.T) {
		statusCode = http.StatusNotFound
		_, err := client.GetUILogs()
		if !errors.Is(err, values.ErrNotFound) {
			t.Fatalf("Expected a not found error but got %v", err)
		}
	})

	t.Run("200", func(t *testing.T) {
		statusCode = http.StatusOK
		uiLogsOut, err := client.GetUILogs()
		if err != nil {
			t.Fatalf("Unexpected error getting UI logs: %v", err)
		}

		if !reflect.DeepEqual(uiLogs.List, uiLogsOut) {
			t.Fatalf("Values do not match:\n%+v\n%+v", uiLogs.List, uiLogsOut)
		}
	})
}

func TestGetSASLLogs(t *testing.T) {
	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(SASLLogsEndpoint.Format("x")), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(http.StatusNotFound, []byte{}, []byte{}, w)
	})
	handlers.Add(http.MethodGet, string(SASLLogsEndpoint.Format("y")), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(http.StatusOK, "THIS IS A LOG", []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("404", func(t *testing.T) {
		_, err := client.GetSASLLogs(context.Background(), "x")
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("200", func(t *testing.T) {
		body, err := client.GetSASLLogs(context.Background(), "y")
		require.NoError(t, err)

		defer body.Close()

		rawLog, err := io.ReadAll(body)
		require.NoError(t, err)

		require.Equal(t, []byte(`"THIS IS A LOG"`), rawLog)
	})
}

func TestGetDiagLog(t *testing.T) {
	var statusCode int

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, "/diag", func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, "THIS IS A LOG", []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("404", func(t *testing.T) {
		statusCode = http.StatusNotFound
		_, err := client.GetDiagLog(context.Background())
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("200", func(t *testing.T) {
		statusCode = http.StatusOK
		body, err := client.GetDiagLog(context.Background())
		require.NoError(t, err)

		defer body.Close()

		rawLog, err := io.ReadAll(body)
		require.NoError(t, err)

		require.Equal(t, []byte(`"THIS IS A LOG"`), rawLog)
	})
}
