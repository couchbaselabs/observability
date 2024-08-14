// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package hazelnut

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
)

type nodeCommon interface {
	UUID() string
	ClusterUUID() string
}

const defaultPort = 9084

type Receiver struct {
	logger      *zap.Logger
	resultStore *store.InMemory
	fileRules   map[string][]rule
	port        int
	node        nodeCommon
}

func NewHazelnut(store *store.InMemory, node nodeCommon) (*Receiver, error) {
	result := &Receiver{
		logger:      zap.L().Named("hazelnut"),
		resultStore: store,
		fileRules:   make(map[string][]rule),
		node:        node,
	}
	rules, err := loadEmbeddedRules()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded rules: %w", err)
	}
	for _, rule := range rules {
		result.fileRules[rule.File] = append(result.fileRules[rule.File], rule)
	}
	return result, nil
}

func (r *Receiver) Port() int {
	return defaultPort
}

func (r *Receiver) Start(ctx context.Context, wg *sync.WaitGroup) error {
	listenConfig := net.ListenConfig{}
	l, err := listenConfig.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", defaultPort))
	if err != nil {
		r.logger.Warn("Failed to listen on default port. Trying random.", zap.Int("defaultPort", defaultPort),
			zap.Error(err))
		l, err = listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}
	r.logger.Info("Listening", zap.String("addr", l.Addr().String()))
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return fmt.Errorf("tried to listen on invalid host: %w", err)
	}
	r.port, err = strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("tried to listen on non-int port: %w", err)
	}

	go func() {
		defer wg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				r.logger.Error("Failed to accept connection", zap.Error(err))
				return
			}
			go r.handleConn(conn)
		}
	}()

	go func() {
		<-ctx.Done()
		r.logger.Warn("Shutting down", zap.Error(ctx.Err()))
		err := l.Close()
		r.logger.Info("Listener closed", zap.Error(err))
	}()

	return nil
}

func (r *Receiver) handleConn(conn net.Conn) {
	err := r.processMessages(conn)
	if err != nil {
		r.logger.Error("Error when processing messages", zap.Error(err))
		conn.Close()
	}
}
