// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"

	sqlite3 "github.com/xeodou/go-sqlcipher"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (db *DB) AddUser(user *values.User) error {
	_, err := db.sqlDB.Exec("INSERT INTO users (user, password, admin) VALUES (?, ?, ?);", user.User, user.Password,
		user.Admin)
	if err != nil {
		if sqlErr, ok := err.(sqlite3.Error); ok {
			if errors.Is(sqlErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return storage.ErrUserAlreadyExists
			}
		}
		return fmt.Errorf("could not add user: %w", err)
	}

	return nil
}

func (db *DB) GetUser(user string) (*values.User, error) {
	result := db.sqlDB.QueryRow("SELECT password, admin FROM users WHERE user = ?;", user)
	returnUser := &values.User{User: user}
	if err := result.Scan(&returnUser.Password, &returnUser.Admin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, values.ErrNotFound
		}

		return nil, fmt.Errorf("could not get user: %w", err)
	}

	return returnUser, nil
}
