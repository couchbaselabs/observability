// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func TestDBAddAndGetUser(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()
	t.Run("new-admin", func(t *testing.T) {
		userIn := &values.User{
			User:     "doc",
			Password: []byte(`password`),
			Admin:    true,
		}
		err := db.AddUser(userIn)
		if err != nil {
			t.Fatalf("Unexpected error adding user: %v", err)
		}

		user, err := db.GetUser("doc")
		if err != nil {
			t.Fatalf("Unexpected error getting user: %v", err)
		}

		if !reflect.DeepEqual(userIn, user) {
			t.Fatalf("Expected %+v fot %+v", userIn, user)
		}
	})

	t.Run("not-unique", func(t *testing.T) {
		err := db.AddUser(&values.User{
			User:     "doc",
			Password: []byte(`password`),
		})
		if err == nil {
			t.Fatalf("Should not be able to insert 2 users with same username")
		}
		require.ErrorIs(t, err, storage.ErrUserAlreadyExists)
	})

	t.Run("not-admin", func(t *testing.T) {
		userIn := &values.User{
			User:     "grumpy",
			Password: []byte(`passw44ord`),
		}
		err := db.AddUser(userIn)
		if err != nil {
			t.Fatalf("Unexpected error adding user: %v", err)
		}

		user, err := db.GetUser("grumpy")
		if err != nil {
			t.Fatalf("Unexpected error getting user: %v", err)
		}

		if !reflect.DeepEqual(userIn, user) {
			t.Fatalf("Expected %+v fot %+v", userIn, user)
		}
	})

	t.Run("user-does-not-exist", func(t *testing.T) {
		_, err := db.GetUser("happy")
		if !errors.Is(err, values.ErrNotFound) {
			t.Fatalf("Expected not found error got: %v", err)
		}
	})
}
