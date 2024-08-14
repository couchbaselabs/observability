// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/auth"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"go.uber.org/zap"
)

type initializeReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func (m *Manager) getInitState(w http.ResponseWriter, _ *http.Request) {
	restutil.SendJSONResponse(http.StatusOK, []byte(fmt.Sprintf(`{"init":%v}`, m.initialized)), w, nil)
}

func (m *Manager) initializeCluster(w http.ResponseWriter, r *http.Request) {
	if m.initialized {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "already initialized",
		}, w, nil)
		return
	}

	var info initializeReq
	if !restutil.DecodeJSONRequestBody(r.Body, &info, w) {
		return
	}

	// validate the request
	if len(info.User) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "user is required",
		}, w, nil)
		return
	}

	if len(info.User) > 64 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "max user length is 64 characters",
		}, w, nil)
		return
	}

	if len(info.Password) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "password is required",
		}, w, nil)
		return
	}

	if len(info.Password) > 64 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "max password length is 64 characters",
		}, w, nil)
		return
	}

	// hash the password
	hashedPassword, err := auth.HashPassword(info.Password)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not hash password",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	err = m.store.AddUser(&values.User{
		User:     info.User,
		Password: hashedPassword,
		Admin:    true,
	})
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not add user",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	m.initialized = true
	zap.S().Infow("(Manager) System initialized")
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) tokenLogin(w http.ResponseWriter, r *http.Request) {
	var info initializeReq
	if !restutil.DecodeJSONRequestBody(r.Body, &info, w) {
		return
	}

	// confirm valid user and password
	userStruct, err := m.store.GetUser(info.User)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "invalid credentials",
				Extras: err.Error(),
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get user",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if !auth.CheckPassword(info.Password, userStruct.Password) {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "invalid credentials",
		}, w, nil)
		return
	}

	raw, err := m.createJWTToken(info.User, time.Hour)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not produce token",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(raw))
}

func (m *Manager) cleanup(w http.ResponseWriter, _ *http.Request) {
	m.janitor.ForceShift()
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) SetupAdminUser(user string, hashedPassword []byte) error {
	err := m.store.AddUser(&values.User{
		User:     user,
		Password: hashedPassword,
		Admin:    true,
	})
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			zap.S().Infow("(Manager) Admin user already exists, auto-provisioned credentials ignored.")
			m.initialized = true
			return nil
		}
		return err
	}

	m.initialized = true
	zap.S().Infow("(Manager) System initialized using auto-provisioned credentials")
	return nil
}
