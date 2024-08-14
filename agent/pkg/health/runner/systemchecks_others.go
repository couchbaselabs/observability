// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

//go:build !linux
// +build !linux

package runner

func getSystemCheckers() map[string]checkerFn {
	return getUniversalCheckers()
}
