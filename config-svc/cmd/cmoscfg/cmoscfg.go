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
	"flag"
	"log"
	"os"

	"github.com/couchbaselabs/observability/config-svc/pkg/api"
	"go.uber.org/zap"
)

var (
	flagHTTPPathPrefix = flag.String("http-path-prefix", "", "URL path to serve the API on")
	flagHTTPHost       = flag.String("http-host", "0.0.0.0", "host to listen on")
	flagHTTPPort       = flag.Int("http-port", 7194, "port to listen on")
	flagDevelopment    = flag.Bool("development", false, "enable development logging and file paths")
)

func main() {
	flag.Parse()

	var (
		baseLogger *zap.Logger
		err        error
	)
	if *flagDevelopment {
		baseLogger, err = zap.NewDevelopment()
	} else {
		baseLogger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatal(err)
	}
	defer baseLogger.Sync()

	logger := baseLogger.Named("main").Sugar()

	if *flagDevelopment {
		if err := os.MkdirAll("./targets", 0o777); err != nil {
			baseLogger.Sugar().Fatalw("Failed to create targets directory", "err", err)
		}
	}

	server, err := api.NewServer(baseLogger, *flagHTTPPathPrefix, !*flagDevelopment)
	if err != nil {
		logger.Fatalw("Failed to create API server", "err", err)
	}
	server.Serve(*flagHTTPHost, *flagHTTPPort)
}
