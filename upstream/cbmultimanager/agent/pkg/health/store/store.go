// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package store

import "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

// InMemory is the store used by the agent to store checker results. The store is volatile and is not expected to
// contain more than a hand-full of items. If the amount of results stores grows to large we will have to migrate to a
// file backed solution.
type InMemory struct {
	checkerResults map[string]*values.WrappedCheckerResult
}

func NewInMemoryStore() *InMemory {
	return &InMemory{checkerResults: make(map[string]*values.WrappedCheckerResult)}
}

func (s *InMemory) SetCheckerResult(name string, result *values.WrappedCheckerResult) {
	s.checkerResults[name] = result
}

func (s *InMemory) GetCheckerResult(name string) (*values.WrappedCheckerResult, error) {
	if res, ok := s.checkerResults[name]; ok {
		return res, nil
	}

	return nil, values.ErrNotFound
}

func (s *InMemory) GetCheckers() map[string]*values.WrappedCheckerResult {
	return s.checkerResults
}

func (s *InMemory) RemoveCheckerResult(name string) {
	delete(s.checkerResults, name)
}
