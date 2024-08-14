// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package config

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
)

func TestDefaultConfig(t *testing.T) {
	app := &cli.App{
		Name:                 "Couchbase health checking agent",
		HelpName:             "cbhealthagent",
		Usage:                "A basic integrated health checking agent for Couchbase Server",
		Version:              meta.Version,
		EnableBashCompletion: true,
		Action: func(cliFlags *cli.Context) error {
			cfg := FromFlags(cliFlags)
			require.Equal(t, cfg.HTTPPort, 9092)
			require.Equal(t, cfg.LogLevel, LogLevel("info"))
			require.Equal(t, cfg.CouchbaseInstallPath, "/opt/couchbase")
			require.Equal(t, cfg.CheckInterval, 10*time.Minute)
			require.Equal(t, cfg.JanitorInterval, 10*time.Minute)
			require.Equal(t, cfg.LogAlertDuration, 1*time.Hour)
			require.Equal(t, cfg.AutoFeatures, true)
			require.Equal(t, cfg.CouchbaseUsername, "Administrator")
			require.Equal(t, cfg.CouchbasePassword, "password")
			return nil
		},
		Flags: Flags,
	}

	os.Args = []string{"package.go"}
	os.Setenv("COUCHBASE_USERNAME", "Administrator")
	os.Setenv("COUCHBASE_PASSWORD", "password")

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func TestUserSetConfig(t *testing.T) {
	app := &cli.App{
		Name:                 "Couchbase health checking agent",
		HelpName:             "cbhealthagent",
		Usage:                "A basic integrated health checking agent for Couchbase Server",
		Version:              meta.Version,
		EnableBashCompletion: true,
		Action: func(cliFlags *cli.Context) error {
			cfg := FromFlags(cliFlags)
			require.Equal(t, cfg.HTTPPort, 65535)
			require.Equal(t, cfg.LogLevel, LogLevel("debug"))
			require.Equal(t, cfg.CouchbaseInstallPath, "/opt")
			require.Equal(t, cfg.CheckInterval, 20*time.Minute)
			require.Equal(t, cfg.JanitorInterval, 20*time.Minute)
			require.Equal(t, cfg.LogAlertDuration, 20*time.Minute)
			require.Equal(t, cfg.AutoFeatures, false)
			require.Equal(t, cfg.CouchbaseUsername, "admin")
			require.Equal(t, cfg.CouchbasePassword, "admin")
			require.Equal(t, cfg.EnableFeatures, []string{"health-agent", "fluent-bit"})
			require.Equal(t, cfg.DisableFeatures, []string{"prometheus-exporter"})
			return nil
		},
		Flags: Flags,
	}

	os.Args = []string{
		"package.go",
		"--http.port", "65535",
		"--log.level", "debug",
		"--couchbase.install-path", "/opt",
		"--couchbase.username", "admin",
		"--couchbase.password", "admin",
		"--features.auto=false", // make sure to assign features.auto using (=)
		"--features.enable", "health-agent,fluent-bit",
		"--features.disable", "prometheus-exporter",
		"--health-agent.check-interval", "20m",
		"--log-analyzer.janitor-interval", "20m",
		"--log-analyzer.alert-duration", "20m",
	}
	os.Setenv("COUCHBASE_USERNAME", "Administrator")
	os.Setenv("COUCHBASE_PASSWORD", "password")

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
