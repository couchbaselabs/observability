package main

import (
	"errors"
	"flag"
	"github.com/couchbaselabs/observability/config-svc/pkg/api"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"github.com/go-playground/validator/v10"
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

	cfg, err := metacfg.ReadConfigFromFile(*flagConfigLocation, true, true)
	if err != nil {
		var validationErr validator.ValidationErrors
		if errors.As(err, &validationErr) {
			var errs []string
			for _, e := range validationErr {
				errs = append(errs, e.Error())
			}
			logger.Fatalw("Invalid configuration", "configLocation", *flagConfigLocation, "errors", errs)
		} else {
			logger.Fatalw("Failed to read configuration", "err", err, "configLocation", *flagConfigLocation)
		}
	}

	server, err := api.NewServer(baseLogger, cfg)
	if err != nil {
		logger.Fatalw("Failed to create API server", "err", err)
	}
	server.Serve()
}
