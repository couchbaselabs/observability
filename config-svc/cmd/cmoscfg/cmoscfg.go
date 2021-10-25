// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"log"

	"github.com/couchbaselabs/observability/config-svc/pkg/api"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var (
	flagConfigLocation = flag.String("config-path", "./config.yaml", "path to read/store the configuration")
	flagHTTPPathPrefix = flag.String("http-path-prefix", "", "URL path to serve the API on")
	flagHTTPHost       = flag.String("http-host", "0.0.0.0", "host to listen on")
	flagHTTPPort       = flag.Int("http-port", 7194, "port to listen on")
)

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

	server, err := api.NewServer(baseLogger, cfg, *flagHTTPPathPrefix)
	if err != nil {
		logger.Fatalw("Failed to create API server", "err", err)
	}
	server.Serve(*flagHTTPHost, *flagHTTPPort)
}
