package couchbase

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/stretchr/testify/require"
)

func TestGetMetrics(t *testing.T) {
	var statusCode int
	sample := json.RawMessage(`{"data":{"result":[{"metric":{"__name__":"sysproc_mem_size","category":
	"system-processes","instance":"ns_server","job":"general","name":"sysproc_mem_size","proc":"babysitter"},
	"values":[[1618503000,"5583470592"],[1618506000,"5583470592"]]}]}}`)

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(PrometheusQueryEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, sample, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("404", func(t *testing.T) {
		statusCode = 404
		_, err := client.GetMetric("2021-04-15T14:00:00.00Z", "2021-04-15T20:00:00.00Z", "sys_cpu_utilization_rate",
			"10m")
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("200", func(t *testing.T) {
		statusCode = 200
		outmetric, err := client.GetMetric("2021-04-15T14:00:00.00Z", "2021-04-15T20:00:00.00Z", "sys_cpu_utilization_rate",
			"10m")
		require.NoError(t, err)

		expected := &Metric{
			Name:     "sysproc_mem_size",
			Category: "system-processes",
			Instance: "ns_server",
			Job:      "general",
			Values: []MetricVal{
				{
					Timestamp: time.Unix(1618503000, 0),
					Value:     "5583470592",
				},
				{
					Timestamp: time.Unix(1618506000, 0),
					Value:     "5583470592",
				},
			},
		}

		require.Equal(t, expected, outmetric)
	})
}
