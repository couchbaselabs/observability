// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

//go:build noui
// +build noui

package ui

import "embed"

var EmbeddedUI embed.FS

const EmbedPresent = false
