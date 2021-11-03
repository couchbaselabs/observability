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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/kennygrant/sanitize"

	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
)

const (
	prometheusTargetsPathDevelopment = "./targets/%s.json"
	prometheusTargetsPathProduction  = "/etc/prometheus/couchbase/custom/%s.json"
)

type prometheusTargets []struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func (s *Server) PostAddPrometheusTarget(ctx echo.Context) error {
	var data v1.PostAddPrometheusTargetJSONBody
	if err := ctx.Bind(&data); err != nil {
		return err
	}

	labels := data.Labels.AdditionalProperties
	if data.NameLabel != nil {
		labels[*data.NameLabel] = data.Name
	}
	value := prometheusTargets{
		{
			Targets: data.Targets,
			Labels:  labels,
		},
	}
	if value[0].Labels == nil {
		value[0].Labels = make(map[string]string)
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var path string
	if s.production {
		path = prometheusTargetsPathProduction
	} else {
		path = prometheusTargetsPathDevelopment
	}
	path = fmt.Sprintf(path, sanitize.Name(data.Name))

	if data.Overwrite == nil || !*data.Overwrite {
		_, err := os.Stat(path)
		if err == nil {
			return echo.NewHTTPError(http.StatusConflict, "target already exists")
		} else if errors.Is(err, fs.ErrNotExist) {
			// Ignore it
		} else {
			return err
		}
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o664)
	if err != nil {
		return err
	}
	if _, err = outFile.Seek(0, 0); err != nil {
		return err
	}
	if _, err = outFile.Write(valueJSON); err != nil {
		return err
	}
	if err = outFile.Close(); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func (s *Server) GetOpenapiJson(ctx echo.Context) error { //nolint:revive
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}
