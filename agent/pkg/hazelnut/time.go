// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// flbTime is an alias of time.Time to parse the msgpack representation of Fluent Bit times.
type flbTime time.Time

func (f *flbTime) MarshalMsgpack() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (f *flbTime) UnmarshalMsgpack(bytes []byte) error {
	// FLB times are 8 bytes, consisting of 4 for the seconds part and 4 for the nanoseconds
	if len(bytes) != 8 {
		return fmt.Errorf("invalid flbTime (expected 8 bytes, got %d)", len(bytes))
	}
	secs := binary.BigEndian.Uint32(bytes[:4])
	nsecs := binary.BigEndian.Uint32(bytes[4:])
	*f = flbTime(time.Unix(int64(secs), int64(nsecs)))
	return nil
}

func init() {
	nil := flbTime(time.Time{})
	msgpack.RegisterExt(0, &nil)
}
