package couchbase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
)

// GetMetric collects specific metrics from prometheus query API in 7.0.0+.
func (c *Client) GetMetric(start, end, metricName, step string) (*Metric, error) {
	params := url.Values{}
	params.Set("query", metricName)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)

	var notFound *cbrest.EndpointNotFoundError
	res, err := c.internalClient.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           PrometheusQueryEndpoint,
		Service:            cbrest.ServiceManagement,
		ExpectedStatusCode: http.StatusOK,
		QueryParameters:    params,
	})

	if errors.As(err, &notFound) {
		return nil, values.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not retrieve metric: %w", err)
	}

	out, err := parseMetric(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not parse metric: %w", err)
	}

	return out, nil
}

func parseMetric(body []byte) (*Metric, error) {
	var overlay struct {
		Data struct {
			Result []struct {
				Metric struct {
					Name     string `json:"name"`
					Category string `json:"category"`
					Instance string `json:"instance"`
					Job      string `json:"job"`
				} `json:"metric"`
				Values [][]json.RawMessage `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	var parsedMetric Metric
	err := json.Unmarshal(body, &overlay)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	if len(overlay.Data.Result) == 0 {
		return &parsedMetric, nil
	}

	lastResult := overlay.Data.Result[len(overlay.Data.Result)-1]
	parsedMetric.Name = lastResult.Metric.Name
	parsedMetric.Category = lastResult.Metric.Category
	parsedMetric.Instance = lastResult.Metric.Instance
	parsedMetric.Job = lastResult.Metric.Job

	for _, item := range lastResult.Values {
		var value MetricVal
		if err = json.Unmarshal(item[1], &value.Value); err != nil {
			return nil, fmt.Errorf("could not retrieve value: %w", err)
		}

		// Parse timestamp as unix time.
		timestamp, err := strconv.ParseInt(string(item[0]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse timestamp: %w", err)
		}

		value.Timestamp = time.Unix(timestamp, 0)
		parsedMetric.Values = append(parsedMetric.Values, value)
	}

	return &parsedMetric, nil
}
