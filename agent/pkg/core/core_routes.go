// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/couchbase/tools-common/aprov"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"

	"github.com/gorilla/mux"

	"github.com/couchbase/tools-common/restutil"
)

const AgentActiveHeader = "CB-HealthAgent-Active"

func (a *Agent) registerRoutes(router *mux.Router) {
	r := router.PathPrefix("/agent/api/v1").Subrouter()

	r.Path("/ping").Methods(http.MethodGet).HandlerFunc(a.pingAgent)
	r.Path("/activate").Methods(http.MethodPost).HandlerFunc(a.activate)
}

type PingResponse struct {
	State    AgentState `json:"state"`
	NodeUUID string     `json:"node"`
}

func (a *Agent) pingAgent(w http.ResponseWriter, _ *http.Request) {
	res := PingResponse{
		State: a.state,
	}
	if a.node != nil {
		res.NodeUUID = a.node.UUID()
	}
	restutil.MarshalAndSend(http.StatusOK, &res, w, nil)
}

type ActivateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *Agent) activate(w http.ResponseWriter, r *http.Request) {
	var req ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    fmt.Errorf("failed to decode request: %w", err).Error(),
		}, w, nil)
		return
	}

	// Validate the given credentials. If they're valid, cache them and bring up the agent.
	bootstrapper := bootstrap.NewKnownCredentialsBootstrapper(req.Username, req.Password)
	node, err := bootstrapper.CreateRESTClient()
	if err != nil {
		a.logger.Warnw("Failed to bootstrap", "error", err)
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusUnprocessableEntity,
			Msg:    fmt.Errorf("failed to bootstrap: %w", err).Error(),
		}, w, nil)
		return
	}

	credsFile, err := a.getCredentialsFilePath()
	if err != nil {
		a.logger.Errorw("Failed to determine credentials path", "error", err)
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    fmt.Errorf("failed to determine credentials path: %w", err).Error(),
		}, w, nil)
		return
	}
	if err := writeCredentialsToFile(credsFile, &aprov.Static{
		Username: req.Username,
		Password: req.Password,
	}); err != nil {
		a.logger.Errorw("Failed to save credentials to file", "path", credsFile, "error", err)
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    fmt.Errorf("failed to save credentials: %w", err).Error(),
		}, w, nil)
		return
	}

	// Note: this uses shutdownServices(), not Shutdown(), as the latter would also shut down the HTTP server before
	// the reply to cbmultimanager is sent.
	a.node = node
	a.shutdownServices()
	a.state = AgentReady
	if err := a.startup(); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    fmt.Errorf("failed to start agent: %w", err).Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, "ok", w, nil)
}
