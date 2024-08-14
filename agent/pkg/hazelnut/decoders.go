// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// messageDecoder is a common interface to parse Fluent Bit log data.
type messageDecoder interface {
	// Decode gets the next log message from this messageDecoder.
	// When it successfully reads a message, it should return:
	//
	// * the timestamp
	// * the message wrapper, containing at least `file` and `message` fields
	// * nil
	//
	// If there are no more messages, it should return (time.Time{},  nil, EOF).
	Decode() (time.Time, map[string]interface{}, error)
}

type jsonDecoder struct {
	dec *json.Decoder
}

func (j *jsonDecoder) Decode() (time.Time, map[string]interface{}, error) {
	var result map[string]interface{}
	err := j.dec.Decode(&result)
	if err != nil {
		return time.Time{}, nil, err
	}
	tsVal, ok := result["date"]
	if !ok {
		return time.Time{}, nil, fmt.Errorf("invalid payload: no date: %w", err)
	}
	s, ns := math.Modf(tsVal.(float64))
	ts := time.Unix(int64(s), int64(ns*1000000000))
	return ts, result, nil
}

type msgpackDecoder struct {
	dec *msgpack.Decoder
}

func (m *msgpackDecoder) Decode() (time.Time, map[string]interface{}, error) {
	// Fluent Bit's msgpack is a fixarray of a time and a map
	// The time can be either:
	// * a fixext2 (type 0) containing two uint32s - seconds and nanoseconds
	// * a float
	// * or a positive uint64
	// (https://github.com/fluent/fluent-bit/blob/2138cee8f4878733956d42d82f6dcf95f0aa9339/src/flb_time.c#L249)
	data, err := m.dec.DecodeSlice()
	if err != nil {
		return time.Time{}, nil, err
	}
	if len(data) != 2 {
		return time.Time{}, nil, fmt.Errorf("invalid data: invalid array length %d", len(data))
	}
	record, ok := data[1].(map[string]interface{})
	if !ok {
		return time.Time{}, nil, fmt.Errorf("invalid data: second element %T", data[1])
	}
	switch ts := data[0].(type) {
	case *flbTime:
		return time.Time(*ts), record, nil
	case float64:
		secs := math.Floor(ts)
		nsecs := (ts - secs) * 1_000_000_000
		return time.Unix(int64(secs), int64(nsecs)), record, nil
	case uint64:
		return time.Unix(int64(ts), 0), record, nil
	default:
		return time.Time{}, nil, fmt.Errorf("invalid data: unknown time type %T", data[0])
	}
}
