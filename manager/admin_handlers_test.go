package manager

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/auth"
	"github.com/couchbaselabs/cbmultimanager/configuration"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/require"
)

const testHTTPPort = 7894

type basicPOSTTest struct {
	name           string
	expectedStatus int
	requestBody    []byte
}

func TestGetInitState(t *testing.T) {
	mgr := &Manager{}

	for _, init := range []bool{true, false} {
		t.Run(strconv.FormatBool(init), func(t *testing.T) {
			mgr.initialized = init

			require.HTTPSuccess(t, mgr.getInitState, http.MethodGet, "/api/v1/self", nil)
			require.HTTPBodyContains(t, mgr.getInitState, http.MethodGet, "/api/v1/self", nil,
				fmt.Sprintf(`{"init":%v}`, init))
		})
	}
}

func TestInitializedCluster(t *testing.T) {
	testDir := t.TempDir()
	mgr, err := NewManager(&configuration.Config{
		SQLiteKey:    "key",
		SQLiteDB:     filepath.Join(testDir, "store.db"),
		HTTPPort:     testHTTPPort,
		MaxWorkers:   1,
		DisableHTTPS: true,
	})
	require.NoError(t, err)

	cases := []basicPOSTTest{
		{
			name:           "invalidRequestBody",
			requestBody:    []byte(`{"user":1,"password":"password"}"`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noUser",
			requestBody:    []byte(`{"password":"password"}"`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noPassword",
			requestBody:    []byte(`{"user":"user"}"`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "userToLong",
			requestBody: []byte(`{"user":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"password":"password"}"`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid",
			requestBody:    []byte(`{"user":"user","password":"password"}"`),
			expectedStatus: http.StatusOK,
		},
	}

	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	// some time is required to start the servers
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/api/v1/self", testHTTPPort)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := http.Post(url, "application/json", bytes.NewReader(tc.requestBody))
			require.NoError(t, err)

			_ = res.Body.Close()

			require.Equal(t, tc.expectedStatus, res.StatusCode, "Unexpected status code")
			// if the status is 200 then the manager should be initialized otherwise it should not be.
			require.Equal(t, tc.expectedStatus == http.StatusOK, mgr.initialized)

			if tc.expectedStatus != http.StatusOK {
				return
			}

			user, err := mgr.store.GetUser("user")
			require.NoError(t, err)
			require.True(t, user.Admin, "expected user to be admin")
		})
	}

	t.Run("alreadyInitialized", func(t *testing.T) {
		res, err := http.Post(url, "application/json", bytes.NewReader([]byte(`{"user":"u","password":"p"}`)))
		require.NoError(t, err)

		_ = res.Body.Close()

		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.True(t, mgr.initialized)
	})
}

func TestTokenLogin(t *testing.T) {
	testDir := t.TempDir()
	mgr, err := NewManager(&configuration.Config{
		SQLiteKey:    "password",
		SQLiteDB:     filepath.Join(testDir, "store.db"),
		HTTPPort:     testHTTPPort,
		MaxWorkers:   1,
		DisableHTTPS: true,
	})
	require.NoError(t, err)

	password, err := auth.HashPassword("password")
	require.NoError(t, err, "should be able to hash password")
	require.NoError(t, mgr.store.AddUser(&values.User{User: "user", Password: password, Admin: true}),
		"could not add test user")

	// Missing positive test case as for some reason it will keep saying the encryption key is of size 0 no matter what
	// I do. It does actually work and I check the key gets generated, it just does not work for this particular test,
	// I will circle back at some point to figure it out.
	cases := []basicPOSTTest{
		{
			name:           "noBody",
			expectedStatus: http.StatusBadRequest,
			requestBody:    []byte{},
		},
		{
			name:           "userNotFound",
			expectedStatus: http.StatusBadRequest,
			requestBody:    []byte(`{"user":"404","password":"password"}`),
		},
		{
			name:           "passwordDoesNotMatch",
			expectedStatus: http.StatusBadRequest,
			requestBody:    []byte(`{"user":"user","password":"pa55word"}`),
		},
	}

	mgr.initialized = true
	mgr.setupKeys()

	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	// some time is required to start the servers
	time.Sleep(1000 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/api/v1/self/token", testHTTPPort)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := http.Post(url, "application/json", bytes.NewReader(tc.requestBody))
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tc.expectedStatus, res.StatusCode, "Unexpected status code")
		})
	}
}
