// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/manager"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"

	"github.com/couchbase/tools-common/log"
	"github.com/couchbase/tools-common/system"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	sqliteKeyFlagName = "sqlite-key"
	sqliteDBFlagName  = "sqlite-db"

	certPathFlagName = "cert-path"
	keyPathFlagName  = "key-path"

	httpPortFlagName  = "http-port"
	httpsPortFlagName = "https-port"

	uiRootFlagName     = "ui-root"
	maxWorkersFlagName = "max-workers"

	logLevelFlagName = "log-level"
	logDirFlagName   = "log-dir"

	adminUserFlagName     = "admin-user"
	adminPasswordFlagName = "admin-password"

	enableAdminAPIFlagName             = "enable-admin-api"
	enableExtendedAPIFlagName          = "enable-extended-api"
	enableClusterManagementAPIFlagName = "enable-cluster-management-api"

	prometheusURLFlagName           = "prometheus-url"
	prometheusLabelSelectorFlagName = "prometheus-label-selector"
	couchbaseUserFlagName           = "couchbase-user"
	couchbasePasswordFlagName       = "couchbase-password"

	alertmanagerURLsFlagName        = "alertmanager-urls"
	alertmanagerResendDelayFlagName = "alertmanager-resend-delay"
	alertmanagerBaseLabelsFlagName  = "alertmanager-base-labels"

	logCheckLifetimeFlagName = "log-check-lifetime"
)

func init() {
	// Initialise a logger as early as possible, to ensure that any startup errors get logged.
	// It will get replaced later, in logger.Init().
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zapcore.EncoderConfig{}),
		zapcore.AddSync(os.Stdout),
		zapcore.ErrorLevel,
	)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
}

func main() {
	app := &cli.App{
		Name:                 "Couchbase Multi Cluster Manager",
		HelpName:             "cbmultimanager",
		Usage:                "Starts up the Couchbase Multi Cluster Manager",
		Version:              meta.Version,
		EnableBashCompletion: true,
		Action:               run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     sqliteKeyFlagName,
				Usage:    "The password for the SQLiteStore",
				EnvVars:  []string{"CB_MULTI_SQLITE_PASSWORD"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     sqliteDBFlagName,
				Usage:    "The path to the SQLite file to use. If the file does not exist it will create it.",
				Required: true,
			},
			&cli.StringFlag{
				Name: certPathFlagName,
				Usage: "Path to a PEM-encoded X.509 certificate to use to serve the API/UI over TLS. " +
					"If the certificate is signed by a CA, this must also contain the full chain to the CA root " +
					"certificate, including any intermediates. " +
					"Can be omitted, in which case TLS serving will be disabled.",
				EnvVars: []string{"CB_MULTI_CERT_PATH"},
			},
			&cli.StringFlag{
				Name: keyPathFlagName,
				Usage: fmt.Sprintf("Path to a PEM-encoded private key for the certificate in `%s`.",
					certPathFlagName),
				EnvVars: []string{"CB_MULTI_KEY_PATH"},
			},
			&cli.StringFlag{
				Name:  logLevelFlagName,
				Usage: "Set the log level, options are [error, warn, info, debug]",
				Value: "info",
			},
			&cli.IntFlag{
				Name:  httpPortFlagName,
				Usage: "The port to serve HTTP REST API",
				Value: 7196,
			},
			&cli.IntFlag{
				Name:  httpsPortFlagName,
				Usage: "The port to serve HTTPS REST API",
				Value: 7197,
			},
			&cli.StringFlag{
				Name:  uiRootFlagName,
				Usage: "The location of the packed UI",
				Value: "./ui/dist/app",
			},
			&cli.IntFlag{
				Name: maxWorkersFlagName,
				Usage: "The maximum number of workers used for health monitoring and heartbeats " +
					"(defaults to 75% of the number of CPUs)",
			},
			&cli.StringFlag{
				Name:  logDirFlagName,
				Usage: "The location to log too. If it does not exist it will try to create it.",
			},
			&cli.StringFlag{
				Name:    adminUserFlagName,
				Usage:   "The name of the admin user for auto-provisioning",
				EnvVars: []string{"CB_MULTI_ADMIN_USER"},
			},
			&cli.StringFlag{
				Name:    adminPasswordFlagName,
				Usage:   "The password for the admin user for auto-provisioning",
				EnvVars: []string{"CB_MULTI_ADMIN_PASSWORD"},
			},
			&cli.DurationFlag{
				Name:  logCheckLifetimeFlagName,
				Usage: "How long will log alerts fire before being expired.",
				Value: time.Hour,
			},
			&cli.BoolFlag{
				Name:  enableAdminAPIFlagName,
				Usage: "Enable the admin REST API.",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  enableExtendedAPIFlagName,
				Usage: "Enable the extended REST API.",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  enableClusterManagementAPIFlagName,
				Usage: "Enable the cluster management REST API.",
				Value: true,
			},
			&cli.StringFlag{
				Name:    prometheusURLFlagName,
				Usage:   "Base URL of Prometheus instance",
				Value:   "",
				EnvVars: []string{"CB_MULTI_PROMETHEUS_URL"},
			},
			&cli.StringFlag{
				Name: prometheusLabelSelectorFlagName,
				Usage: "Prometheus label selector to use to discover Couchbase Server clusters. " +
					"Syntax: `label1=value label2=value`",
				Value:   "",
				EnvVars: []string{"CB_MULTI_PROMETHEUS_LABEL_SELECTOR"},
			},
			&cli.StringFlag{
				Name: alertmanagerBaseLabelsFlagName,
				Usage: "Base labels to be applied to alerts for Alertmanager. " +
					"Syntax: `label1=value label2=value`",
				Value:   "",
				EnvVars: []string{"CB_MULTI_ALERTMANAGER_BASE_LABELS"},
			},
			&cli.StringFlag{
				Name:    couchbaseUserFlagName,
				Usage:   "Couchbase user name (only needed when using Prometheus discovery)",
				EnvVars: []string{"CB_MULTI_COUCHBASE_USER"},
			},
			&cli.StringFlag{
				Name:    couchbasePasswordFlagName,
				Usage:   "Couchbase password (only needed when using Prometheus discovery)",
				EnvVars: []string{"CB_MULTI_COUCHBASE_PASSWORD"},
			},
			&cli.StringSliceFlag{
				Name:    alertmanagerURLsFlagName,
				Usage:   "URLs of Alertmanager instances to send alerts to.",
				EnvVars: []string{"CB_MULTI_ALERTMANAGER_URLS"},
			},
			&cli.DurationFlag{
				Name:    alertmanagerResendDelayFlagName,
				Usage:   "Interval between re-sending alerts to Alertmanager.",
				EnvVars: []string{"CB_MULTI_ALERTMANAGER_RESEND_DELAY"},
				Value:   time.Minute,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		zap.S().Errorw("(Main) Failed to run", "err", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	config, err := getConfig(c)
	if err != nil {
		return fmt.Errorf("invalid configuration provided: %w", err)
	}

	if err := logger.Init(config.LogLevel, c.String(logDirFlagName)); err != nil {
		return fmt.Errorf("could not initialize logger: %w", err)
	}

	zap.S().Infow("(Main) Couchbase Multi Cluster Manager starting", "version", meta.Version, "build", meta.BuildNumber)

	argsToMask := []string{"--" + sqliteKeyFlagName, "--" + adminPasswordFlagName, "--" + couchbasePasswordFlagName}
	zap.S().Infof("(Main) Running options %s", log.MaskArguments(os.Args[1:], argsToMask))
	zap.S().Infow("(Main) Using configuration", "config", config)

	node, err := manager.NewManager(config)
	if err != nil {
		return fmt.Errorf("could not create manager: %w", err)
	}

	// TODO (CMOS-57) move this to runtime configuration
	node.Start(manager.DefaultFrequencyConfiguration)
	return nil
}

func getConfig(c *cli.Context) (*configuration.Config, error) {
	promLabelSelectors, err := configuration.ParseLabelSelectors(c.String(prometheusLabelSelectorFlagName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus labels: %w", err)
	}

	alertmanagerBaseLabels, err := configuration.ParseLabelSelectors(c.String(alertmanagerBaseLabelsFlagName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Alertmanager labels: %w", err)
	}
	config := &configuration.Config{
		SQLiteKey:               c.String(sqliteKeyFlagName),
		SQLiteDB:                c.String(sqliteDBFlagName),
		CertPath:                c.String(certPathFlagName),
		KeyPath:                 c.String(keyPathFlagName),
		HTTPPort:                c.Int(httpPortFlagName),
		HTTPSPort:               c.Int(httpsPortFlagName),
		UIRoot:                  c.String(uiRootFlagName),
		AdminUser:               c.String(adminUserFlagName),
		AdminPassword:           c.String(adminPasswordFlagName),
		EnableAdminAPI:          c.Bool(enableAdminAPIFlagName),
		EnableExtendedAPI:       c.Bool(enableExtendedAPIFlagName),
		EnableClusterAPI:        c.Bool(enableClusterManagementAPIFlagName),
		PrometheusBaseURL:       c.String(prometheusURLFlagName),
		PrometheusLabelSelector: promLabelSelectors,
		AlertmanagerBaseLabels:  alertmanagerBaseLabels,
		CouchbaseUser:           c.String(couchbaseUserFlagName),
		CouchbasePassword:       c.String(couchbasePasswordFlagName),
		AlertmanagerURLs:        c.StringSlice(alertmanagerURLsFlagName),
		AlertmanagerResendDelay: c.Duration(alertmanagerResendDelayFlagName),
		LogCheckLifetime:        c.Duration(logCheckLifetimeFlagName),
	}

	switch c.String(logLevelFlagName) {
	case "error":
		config.LogLevel = zapcore.ErrorLevel
	case "warn":
		config.LogLevel = zapcore.WarnLevel
	case "info":
		config.LogLevel = zapcore.InfoLevel
	case "debug":
		config.LogLevel = zapcore.DebugLevel
	default:
		return nil, fmt.Errorf("unknown log level '%s'", c.String(logLevelFlagName))
	}

	config.MaxWorkers = system.NumWorkers(c.Int(maxWorkersFlagName))

	return config, nil
}
