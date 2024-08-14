// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package core

import (
	"path/filepath"
	"testing"

	"github.com/couchbase/tools-common/aprov"
	"github.com/stretchr/testify/require"
)

func TestCredentialsFileRoundTrip(t *testing.T) {
	tmpdir := t.TempDir()
	testFile := filepath.Join(tmpdir, "cbhealthagent.pw")
	testCreds := &aprov.Static{
		UserAgent: "test",
		Username:  "Administrator",
		Password:  "password",
	}

	require.NoError(t, writeCredentialsToFile(testFile, testCreds))

	creds, err := readCredentialsFromFile(testFile)
	require.NoError(t, err)
	require.Equal(t, testCreds, creds)
}
