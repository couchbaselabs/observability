// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package server

import (
	"net/http"
	"sync"
)

// SwitchHandler is a http.Handler that allows replacing the handler at runtime.
type SwitchHandler struct {
	handler http.Handler
	mux     sync.RWMutex
}

func (s *SwitchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.RLock()
	h := s.handler
	s.mux.RUnlock()
	h.ServeHTTP(w, r)
}

func (s *SwitchHandler) ReplaceHandler(h http.Handler) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.handler = h
}
