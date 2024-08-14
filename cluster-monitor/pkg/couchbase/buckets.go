// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (c *Client) GetPoolsBucket() ([]values.Bucket, error) {
	res, err := c.get(PoolsBucketEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get pools buckets data: %w", err)
	}

	var buckets []values.Bucket
	if err = json.Unmarshal(res.Body, &buckets); err != nil {
		return nil, fmt.Errorf("could not unmarshal buckets: %w", err)
	}

	return buckets, err
}

func (c *Client) GetBucketsSummary() (values.BucketsSummary, error) {
	res, err := c.get(PoolsBucketEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get pools buckets data: %w", err)
	}

	return values.MarshallBucketsSummaryFromRest(bytes.NewReader(res.Body))
}

func (c *Client) GetBucketStats(bucketName string) (*values.BucketStat, error) {
	res, err := c.get(PoolsBucketStatsEndpoint.Format(bucketName))
	if err != nil {
		return nil, fmt.Errorf("could not get pools bucket stats data: %w", err)
	}

	var overlay struct {
		Op struct {
			Samples struct {
				VbActiveRatio []float64 `json:"vb_active_resident_items_ratio"`
				MemUsed       []float64 `json:"mem_used"`
			}
		}
	}

	if err = json.Unmarshal(res.Body, &overlay); err != nil {
		return nil, fmt.Errorf("couldn't retrieve bucket stats: %w", err)
	}
	stats := &values.BucketStat{
		VbActiveRatio: overlay.Op.Samples.VbActiveRatio,
		MemUsed:       overlay.Op.Samples.MemUsed,
	}

	return stats, nil
}
