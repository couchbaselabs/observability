package main

import (
	"fmt"
	"os"

	"github.com/couchbaselabs/cbmultimanager/configuration"
	"github.com/couchbaselabs/cbmultimanager/logger"
	"github.com/couchbaselabs/cbmultimanager/manager"

	"github.com/couchbase/tools-common/log"
	"github.com/couchbase/tools-common/system"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	app := &cli.App{
		Name:                 "Couchbase Multi Cluster Manager",
		HelpName:             "cbmultimanager",
		Usage:                "Starts up the Couchbase Multi Cluster Manager",
		Version:              "0.0.1",
		EnableBashCompletion: true,
		Action:               run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "sqlite-key",
				Usage:    "The password for the SQLiteStore",
				EnvVars:  []string{"CB_MULTI_SQLITE_PASSWORD"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "sqlite-db",
				Usage:    "The path to the SQLite file to use. If the file does not exist it will create it.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "cert-path",
				Usage:    "The certificate path to use for TLS",
				EnvVars:  []string{"CB_MULTI_CERT_PATH"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "key-path",
				Usage:    "The path to the key",
				EnvVars:  []string{"CB_MULTI_KEY_PATH"},
				Required: true,
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the log level, options are [error, warn, info, debug]",
				Value: "info",
			},
			&cli.IntFlag{
				Name:  "http-port",
				Usage: "The port to serve HTTP REST API",
				Value: 7196,
			},
			&cli.IntFlag{
				Name:  "https-port",
				Usage: "The port to serve HTTPS REST API",
				Value: 7197,
			},
			&cli.StringFlag{
				Name:  "ui-root",
				Usage: "The location of the packed UI",
				Value: "./ui/app/dist/app",
			},
			&cli.IntFlag{
				Name: "max-workers",
				Usage: "The maximum number of workers used for health monitoring and heartbeats " +
					"(defaults to 75% of the number of CPUs)",
			},
			&cli.StringFlag{
				Name:  "log-dir",
				Usage: "The location to log too. If it does not exist it will try to create it.",
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

	if err := logger.Init(config.LogLevel, c.String("log-dir")); err != nil {
		return fmt.Errorf("could not initialize logger: %w", err)
	}

	zap.S().Infof("(Main) Running options %s", log.MaskArguments(os.Args[1:], []string{"--sqlite-key"}))
	zap.S().Infof("(Main) Maximum workers set to %d", config.MaxWorkers)

	node, err := manager.NewManager(config)
	if err != nil {
		return fmt.Errorf("could not create manager: %w", err)
	}

	node.Start(manager.DefaultFrequencyConfiguration)
	return nil
}

func getConfig(c *cli.Context) (*configuration.Config, error) {
	config := &configuration.Config{
		SQLiteKey: c.String("sqlite-key"),
		SQLiteDB:  c.String("sqlite-db"),
		CertPath:  c.String("cert-path"),
		KeyPath:   c.String("key-path"),
		HTTPPort:  c.Int("http-port"),
		HTTPSPort: c.Int("https-port"),
		UIRoot:    c.String("ui-root"),
	}

	switch c.String("log-level") {
	case "error":
		config.LogLevel = zapcore.ErrorLevel
	case "warn":
		config.LogLevel = zapcore.WarnLevel
	case "info":
		config.LogLevel = zapcore.InfoLevel
	case "debug":
		config.LogLevel = zapcore.DebugLevel
	default:
		return nil, fmt.Errorf("unknown log level '%s'", c.String("log-level"))
	}

	config.MaxWorkers = system.NumWorkers(c.Int("max-workers"))

	return config, nil
}
