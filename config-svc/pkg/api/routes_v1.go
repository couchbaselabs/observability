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

package api

import (
	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) GetConfig(ctx echo.Context) error {
	return ctx.Blob(http.StatusOK, "text/yaml", s.cfg.Get().ToYAML())
}

func (s *Server) GetClusters(ctx echo.Context) error {
	return nil
}

func (s *Server) GetOpenapiJson(ctx echo.Context) error {
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}