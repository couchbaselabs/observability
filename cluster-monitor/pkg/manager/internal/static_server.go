// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type StaticSiteServer struct {
	fs          http.FileSystem
	baseHandler http.Handler
}

// ServeStaticSite serves static files from the given fs, with special handling for index.html and missing paths.
// It behaves identically to http.FileServer, except that instead of returning 404 for an unknown path, it will
// serve the contents of /index.html. This allows a JavaScript router to take over determining what to display.
// It will also return 403 Forbidden for any hidden files.
func ServeStaticSite(fs http.FileSystem) http.Handler {
	return &StaticSiteServer{
		fs:          fs,
		baseHandler: http.FileServer(fs),
	}
}

func (s *StaticSiteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	if strings.HasSuffix(upath, "/") {
		upath += "index.html"
	}
	upath = path.Clean(upath)

	base := filepath.Base(upath)
	if len(base) > 0 && base[0] == '.' {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	file, err := s.fs.Open(upath)

	if errors.Is(err, os.ErrNotExist) {
		// Go special-cases /index.html to redirect to /, which causes a redirect loop.
		r.URL.Path = "/"
		s.baseHandler.ServeHTTP(w, r)
		return
	}

	// Re-close the file, and pass it on to the base handler to serve.
	// Safe to ignore the error here, because http.FileServer will do fs.Open() on it again,
	// and handle the error properly.
	if file != nil {
		file.Close()
	}

	s.baseHandler.ServeHTTP(w, r)
}
