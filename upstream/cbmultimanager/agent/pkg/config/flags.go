// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	flagHTTPPort = "http.port"
	flagLogLevel = "log.level"

	flagCouchbaseInstallPath = "couchbase.install-path"
	flagCouchbaseUsername    = "couchbase.username"
	flagCouchbasePassword    = "couchbase.password"

	flagAutoFeatures    = "features.auto"
	flagEnableFeatures  = "features.enable"
	flagDisableFeatures = "features.disable"

	flagCheckInterval = "health-agent.check-interval"

	flagJanitorInterval  = "log-analyzer.janitor-interval"
	flagLogAlertDuration = "log-analyzer.alert-duration"
)

var Flags = []cli.Flag{
	&cli.IntFlag{
		Name:  flagHTTPPort,
		Usage: "Port that exposes the agent REST API.",
		Value: 9092,
	},
	&cli.StringFlag{
		Name:  flagLogLevel,
		Usage: "Level to log at",
		Value: "info",
	},
	&cli.StringFlag{
		Name:  flagCouchbaseInstallPath,
		Usage: "Path that Couchbase Server is installed at.",
		Value: "/opt/couchbase",
	},
	&cli.StringFlag{
		Name: flagCouchbaseUsername,
		Usage: "Username of a read-only admin user for Couchbase Server. If omitted, " +
			"cbhealthagent will automatically authenticate as an administrator.",
		EnvVars: []string{"COUCHBASE_USERNAME", "COUCHBASE_OPERATOR_USER"},
	},
	&cli.StringFlag{
		Name:    flagCouchbasePassword,
		Usage:   fmt.Sprintf("Password of the %s user.", flagCouchbaseUsername),
		EnvVars: []string{"COUCHBASE_PASSWORD", "COUCHBASE_OPERATOR_PASS"},
	},
	&cli.DurationFlag{
		Name:  flagCheckInterval,
		Usage: "How often to refresh health check data.",
		Value: 10 * time.Minute,
	},
	&cli.DurationFlag{
		Name:  flagJanitorInterval,
		Usage: "How often to clean up stale log alerts.",
		Value: 10 * time.Minute,
	},
	&cli.DurationFlag{
		Name: flagLogAlertDuration,
		Usage: "How long will log alerts fire before they are cleaned up, if no matching message is seen in the " +
			"meantime.",
		Value: 1 * time.Hour,
	},
	&cli.BoolFlag{
		Name:  flagAutoFeatures,
		Usage: "Automatically determine features to activate. Use (=) with value to set/unset flag.",
		Value: true,
	},
	&cli.StringSliceFlag{
		Name:  flagEnableFeatures,
		Usage: fmt.Sprintf("Features to enable. Overrides %s.", flagAutoFeatures),
	},
	&cli.StringSliceFlag{
		Name:  flagDisableFeatures,
		Usage: fmt.Sprintf("Features to disable. Overrides %s.", flagAutoFeatures),
	},
}

func splitFlagValue(slice []string) []string {
	if len(slice) > 0 {
		return strings.Split(slice[0], ",")
	}
	return slice
}

func FromFlags(ctx *cli.Context) Config {
	return Config{
		HTTPPort:             ctx.Int(flagHTTPPort),
		LogLevel:             LogLevel(ctx.String(flagLogLevel)),
		CheckInterval:        ctx.Duration(flagCheckInterval),
		JanitorInterval:      ctx.Duration(flagJanitorInterval),
		LogAlertDuration:     ctx.Duration(flagLogAlertDuration),
		AutoFeatures:         ctx.Bool(flagAutoFeatures),
		EnableFeatures:       splitFlagValue(ctx.StringSlice(flagEnableFeatures)),
		DisableFeatures:      splitFlagValue(ctx.StringSlice(flagDisableFeatures)),
		CouchbaseUsername:    ctx.String(flagCouchbaseUsername),
		CouchbasePassword:    ctx.String(flagCouchbasePassword),
		CouchbaseInstallPath: ctx.String(flagCouchbaseInstallPath),
	}
}
