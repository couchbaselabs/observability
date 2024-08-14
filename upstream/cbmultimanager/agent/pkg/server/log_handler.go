// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package server

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type LogHandler struct {
	logger *zap.SugaredLogger
	wraps  http.Handler
}

func (l *LogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.wraps.ServeHTTP(w, r)
	end := time.Now()
	l.logger.Debugf("%s %s %s (%s); %v", r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("User-Agent"), end.Sub(start))
}
