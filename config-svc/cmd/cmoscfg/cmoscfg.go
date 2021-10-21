package main

import (
	"flag"
	"github.com/couchbaselabs/observability/config-svc/pkg/api"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"go.uber.org/zap"
	"log"
)

var flagConfigLocation = flag.String("config-path", "./config.yaml", "path to read/store the configuration")

func main() {
	flag.Parse()

	baseLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer baseLogger.Sync()

	logger := baseLogger.Named("main").Sugar()

	cfg, err := metacfg.ReadConfigFromFile(baseLogger.Named("metacfg"), *flagConfigLocation, true, true)
	if err != nil {
		logger.Fatalw("Failed to read configuration", "err", err, "configLocation", *flagConfigLocation)
	}

	server, err := api.NewServer(baseLogger, cfg)
	if err != nil {
		logger.Fatalw("Failed to create API server", "err", err)
	}
	server.Serve()
}
