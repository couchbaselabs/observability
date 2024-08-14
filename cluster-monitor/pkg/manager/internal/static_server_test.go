// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func createTestFile(t *testing.T, filePath, content string, perms os.FileMode) { //nolint:unparam
	testStaticFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testStaticFile.Write([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if err = testStaticFile.Close(); err != nil {
		t.Fatal(err)
	}
}

func createTestFiles(t *testing.T) string {
	tmpDir := t.TempDir()
	createTestFile(t, path.Join(tmpDir, "index.html"), "Index!", 0o774)
	createTestFile(t, path.Join(tmpDir, "test.txt"), "Success!", 0o774)
	createTestFile(t, path.Join(tmpDir, ".hidden.txt"), "Hidden!", 0o774)
	require.NoError(t, os.Mkdir(path.Join(tmpDir, "directory"), 0o777))
	require.NoError(t, os.Mkdir(path.Join(tmpDir, "empty-directory"), 0o777))
	createTestFile(t, path.Join(tmpDir, "directory", "index.html"), "Nested Index!", 0o774)
	return tmpDir
}

func testRequest(router *mux.Router, path string, expectedStatus int, expectedBody string) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != expectedStatus {
			t.Fatalf("wrong status code: expected %d, got %d", expectedStatus, status)
		}
		require.Equal(t, expectedStatus, rr.Code, "unexpected request status")
		if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(expectedBody) {
			t.Fatalf("unexpected body: got %v, want %v", rr.Body.String(), expectedBody)
		}
	}
}

func TestStaticServe(t *testing.T) {
	testRoot := createTestFiles(t)
	handler := ServeStaticSite(http.Dir(testRoot))
	router := mux.NewRouter()
	router.PathPrefix("/monitor").Handler(http.StripPrefix("/monitor", handler))

	t.Run("Root", testRequest(router, "/monitor", http.StatusOK, "Index!"))
	t.Run("RootTrailingSlash", testRequest(router, "/monitor/", http.StatusOK, "Index!"))

	t.Run("HiddenFile", testRequest(router, "/monitor/.hidden.txt", http.StatusForbidden, "forbidden"))
	t.Run("Directory", testRequest(router, "/monitor/directory", http.StatusMovedPermanently, ""))
	t.Run("DirectoryTrailingSlash", testRequest(router, "/monitor/directory/", http.StatusOK, "Nested Index!"))

	t.Run("StaticFile", testRequest(router, "/monitor/test.txt", http.StatusOK, "Success!"))

	t.Run("NonExistentFile", testRequest(router, "/monitor/non-existent", http.StatusOK, "Index!"))
}
