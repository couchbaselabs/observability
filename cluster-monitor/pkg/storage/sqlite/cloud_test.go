// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestAddGetCloudCredentials(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("missing-data", func(t *testing.T) {
		require.Error(t, db.AddCloudCredentials(&values.Credential{Name: "creds-0"}))
	})

	t.Run("get-empty", func(t *testing.T) {
		creds, err := db.GetCloudCredentials(true)
		require.NoError(t, err)
		require.Len(t, creds, 0)
	})

	t.Run("valid", func(t *testing.T) {
		cred := &values.Credential{
			Name:      "creds-0",
			AccessKey: "a",
			SecretKey: "b",
		}

		require.NoError(t, db.AddCloudCredentials(cred))

		creds, err := db.GetCloudCredentials(true)
		require.NoError(t, err)
		require.Len(t, creds, 1)

		cred.DateAdded = creds[0].DateAdded
		require.Equal(t, cred, creds[0])
		require.NotZero(t, cred.DateAdded)
	})

	t.Run("repeated-name", func(t *testing.T) {
		require.Error(t, db.AddCloudCredentials(&values.Credential{Name: "creds-0", AccessKey: "a", SecretKey: "b"}))
	})
}
