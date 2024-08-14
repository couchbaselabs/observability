// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (db *DB) AddAlias(alias *values.ClusterAlias) error {
	_, err := db.sqlDB.Exec("INSERT INTO aliases (alias, clusterUUID) VALUES (?, ?);", alias.Alias, alias.ClusterUUID)
	if err != nil {
		return fmt.Errorf("could not add alias '%s' -> '%s': %w", alias.Alias, alias.ClusterUUID, err)
	}

	return nil
}

func (db *DB) DeleteAlias(alias string) error {
	_, err := db.sqlDB.Exec("DELETE FROM aliases WHERE alias = ?;", alias)
	if err != nil {
		return fmt.Errorf("could not delete alias '%s': %w", alias, err)
	}

	return nil
}

func (db *DB) GetAlias(alias string) (*values.ClusterAlias, error) {
	row := db.sqlDB.QueryRow("SELECT clusterUUID FROM aliases WHERE alias = ?;", alias)

	clusterAlias := &values.ClusterAlias{Alias: alias}
	if err := row.Scan(&clusterAlias.ClusterUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, values.ErrNotFound
		}

		return nil, fmt.Errorf("could not scan alias: %w", err)
	}

	return clusterAlias, nil
}
