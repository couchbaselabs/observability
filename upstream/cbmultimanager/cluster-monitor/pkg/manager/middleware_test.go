// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/auth"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func okFun(w http.ResponseWriter, _ *http.Request) {
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func TestInitializedMiddleware(t *testing.T) {
	manager := &Manager{
		initialized: false,
	}

	// create a router for the test
	router := mux.NewRouter()
	router.Use(manager.initializedMiddleware)
	router.HandleFunc("/api/v1/clusters", okFun)
	router.HandleFunc("/api/v1/self", okFun)

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	type testCase struct {
		name           string
		url            string
		initialized    bool
		expectedStatus int
	}

	cases := []testCase{
		{
			name:           "not-initialized",
			url:            "/api/v1/clusters",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "initialized",
			url:            "/api/v1/clusters",
			initialized:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not-initialized-init-endpoint",
			url:            "/api/v1/self",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "initialized-init-endpoint",
			url:            "/api/v1/self",
			initialized:    true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager.initialized = tc.initialized
			resp, err := http.Get(testServer.URL + tc.url)
			require.Nil(t, err, "Should be able to do the request")
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "Unexpected status code")
		})
	}
}

func TestAuthMiddlewareNoAuth(t *testing.T) {
	testDir := t.TempDir()
	manager, err := NewManager(&configuration.Config{
		SQLiteKey: "password",
		SQLiteDB:  filepath.Join(testDir, "store.db"),
	})
	require.Nil(t, err, "Could not create manager")

	noAuthEndpoints := []string{"/", PathUIRoot, "/api/v1/self", "/api/v1/self/token"}
	// create a router for the test
	router := mux.NewRouter()
	router.Use(manager.authMiddleware)
	for _, endpoint := range noAuthEndpoints {
		router.HandleFunc(endpoint, okFun)
	}

	router.HandleFunc("/api/v1/authed", okFun)

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	for _, endpoints := range noAuthEndpoints {
		t.Run(endpoints, func(t *testing.T) {
			res, err := http.Get(testServer.URL + endpoints)
			require.Nil(t, err, "Expected to be able to do the request")
			defer res.Body.Close()

			require.Equal(t, res.StatusCode, http.StatusOK)
		})
	}

	t.Run("auth-require-endpoint", func(t *testing.T) {
		res, err := http.Get(testServer.URL + "/api/v1/authed")
		require.Nil(t, err, "Expected to be able to do the request")
		defer res.Body.Close()

		require.Equal(t, res.StatusCode, http.StatusUnauthorized)
	})
}

func TestAuthMiddlewareBasic(t *testing.T) {
	testDir := t.TempDir()
	manager, err := NewManager(&configuration.Config{
		SQLiteKey: "password",
		SQLiteDB:  filepath.Join(testDir, "store.db"),
	})
	require.Nil(t, err, "Could not create manager")

	password, err := auth.HashPassword("password")
	require.Nil(t, err, "Could not hash password")
	require.Nil(t, manager.store.AddUser(&values.User{User: "user", Password: password, Admin: true}),
		"could not add test user")

	// create a router for the test
	router := mux.NewRouter()
	router.Use(manager.authMiddleware)
	router.HandleFunc("/api/v1/authed", okFun)

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	type testCase struct {
		name           string
		user           string
		password       string
		expectedStatus int
	}

	cases := []testCase{
		{
			name:           "valid-auth",
			user:           "user",
			password:       "password",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid-user-bad-password",
			user:           "user",
			password:       "password1",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid-user",
			user:           "user1",
			password:       "password",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, testServer.URL+"/api/v1/authed", nil)
			require.Nil(t, err, "Expected to be able to create request")

			request.SetBasicAuth(tc.user, tc.password)

			res, err := client.Do(request)
			require.Nil(t, err, "Expected to be able to do the request")
			defer res.Body.Close()

			require.Equal(t, res.StatusCode, tc.expectedStatus)
		})
	}
}

func TestAuthMiddlewareJWT(t *testing.T) {
	testDir := t.TempDir()
	manager, err := NewManager(&configuration.Config{
		SQLiteKey: "password",
		SQLiteDB:  filepath.Join(testDir, "store.db"),
	})
	require.Nil(t, err, "Could not create manager")

	password, err := auth.HashPassword("password")
	require.Nil(t, err, "Could not hash password")
	require.Nil(t, manager.store.AddUser(&values.User{User: "user", Password: password, Admin: true}),
		"could not add test user")

	// create a router for the test
	router := mux.NewRouter()
	router.Use(manager.authMiddleware)
	router.HandleFunc("/api/v1/authed", okFun)

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	type testCase struct {
		name           string
		token          string
		expectedStatus int
	}

	manager.setupKeys()
	validToken, err := manager.createJWTToken("user", time.Hour)
	require.Nil(t, err, "Could not create valid token")

	expiredToken, err := manager.createJWTToken("user", -time.Hour)
	require.Nil(t, err, "Could not create expired token")

	invalidUserToken, err := manager.createJWTToken("user1", time.Hour)
	require.Nil(t, err, "Could not create token for invalid user")

	invalidToken, err := manager.createJWTToken("user", time.Hour)
	require.Nil(t, err, "Could not create token for invalid user")

	invalidToken = "garbage" + invalidToken

	cases := []testCase{
		{
			name:           "valid-auth",
			expectedStatus: http.StatusOK,
			token:          validToken,
		},
		{
			name:           "expired-token",
			expectedStatus: http.StatusUnauthorized,
			token:          expiredToken,
		},
		{
			name:           "invalid-user",
			expectedStatus: http.StatusUnauthorized,
			token:          invalidUserToken,
		},
		{
			name:           "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			token:          invalidToken,
		},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, testServer.URL+"/api/v1/authed", nil)
			require.Nil(t, err, "Expected to be able to create request")

			request.Header.Set("Authorization", "Bearer "+tc.token)

			res, err := client.Do(request)
			require.Nil(t, err, "Expected to be able to do the request")
			defer res.Body.Close()

			require.Equal(t, res.StatusCode, tc.expectedStatus)
		})
	}
}
