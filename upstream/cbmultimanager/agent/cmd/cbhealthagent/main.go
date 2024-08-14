// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/config"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
)

func main() {
	app := &cli.App{
		Name:                 "Couchbase health checking agent",
		HelpName:             "cbhealthagent",
		Usage:                "A basic integrated health checking agent for Couchbase Server",
		Version:              meta.Version,
		EnableBashCompletion: true,
		Action:               run,
		Flags:                config.Flags,
	}

	if err := app.Run(os.Args); err != nil {
		zap.S().Errorw("(Main) could not run health agent", "err", err)
		os.Exit(1)
	}
}

func run(cliCtx *cli.Context) error {
	cfg := config.FromFlags(cliCtx)
	// Not using the file configuration so it will never return an error.
	_ = logger.Init(cfg.LogLevel.ToZap(), "")

	zap.S().Infof("(Main) Running options %v", os.Args)

	agent, err := core.CreateAgent(cfg)
	if err != nil {
		zap.S().Fatalw("(Main) Failed to create agent!", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := agent.Start(ctx); err != nil {
		zap.S().Fatalw("(Main) Failed to start agent!", "error", err)
	}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)

	// CMOS-345: need to know if services have failed
	<-sigCh
	zap.S().Info("(Main) Got interrupt signal, starting graceful shutdown")
	cancel()

	agent.Shutdown()

	select {
	case <-agent.Done():
		zap.S().Info("(Main) Thank you and goodnight!")
		os.Exit(0)
	case <-sigCh:
		zap.S().Warn("(Main) Got second interrupt, exiting immediately!")
		os.Exit(1)
	}

	return nil
}
