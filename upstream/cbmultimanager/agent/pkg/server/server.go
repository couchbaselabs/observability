// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
)

type Server struct {
	store   *store.InMemory
	port    int
	handler *SwitchHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewServer(store *store.InMemory, port int, router *mux.Router) *Server {
	return &Server{store: store, port: port, handler: &SwitchHandler{handler: router}}
}

func (s *Server) ReplaceRouter(r *mux.Router) {
	s.handler.ReplaceHandler(r)
}

func (s *Server) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	srv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", s.port), Handler: &LogHandler{
		logger: zap.S().Named("HTTP"),
		wraps:  s.handler,
	}}

	zap.S().Infow("(Server) Starting", "port", s.port)

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		zap.S().Errorw("(Server) Failed to listen", "err", err)
		return err
	}

	go func(l net.Listener) {
		defer wg.Done()
		if err := srv.Serve(l); err != nil {
			zap.S().Warnw("(Server) Server stopped", "err", err)
		}
	}(l)

	go func() {
		<-s.ctx.Done()

		zap.S().Infow("(Server) Shutting down health endpoint")

		innerCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		if err := srv.Shutdown(innerCtx); err != nil {
			zap.S().Errorw("(Server) Issue gracefully shutting down health endpoint", "err", err)
		}
	}()

	return nil
}
