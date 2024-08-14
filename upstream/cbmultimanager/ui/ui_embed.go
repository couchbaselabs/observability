// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

//go:build !noui
// +build !noui

package ui

import (
	"embed"
	"io/fs"
)

// With this go:embed line, index.html will really be called `dist/app/index.html`, hence the `fs.Sub` below to ensure
// that, when the router tries to serve it, it can resolve `/index.html`.

//go:embed dist/app
var embeddedUI embed.FS

func mustFS(fsys fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return fsys
}

var EmbeddedUI = mustFS(fs.Sub(embeddedUI, "dist/app"))

const EmbedPresent = true
