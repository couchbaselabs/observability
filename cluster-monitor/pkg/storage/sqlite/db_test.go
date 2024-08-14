// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	_ "github.com/xeodou/go-sqlcipher"
)

func createEmptyDB(t *testing.T) (storage.Store, string) {
	db, path := createEmptyDBOnVersion0(t)
	require.NoError(t, setupNewSQLDB(db, "key", true))
	return db, path
}

func createEmptyDBOnVersion0(t *testing.T) (*DB, string) {
	testDir := t.TempDir()

	filePath := filepath.Join(testDir, "db.sqlite")

	sqlDb, err := sql.Open("sqlite3", fmt.Sprintf("%s?_key=%s", filePath, "key"))
	require.NoError(t, err)

	db := &DB{
		fileName: filePath,
		sqlDB:    sqlDb,
	}
	return db, filePath
}

func TestNewSQLiteDBFileDoesNotExist(t *testing.T) {
	db, filePath := createEmptyDB(t)

	err := db.Close()
	if err != nil {
		t.Fatalf("Could not close the database: %v", err)
	}

	if _, err = os.Stat(filePath); err != nil {
		t.Fatalf("Database file does not exist: %v", err)
	}

	// verify that the file was created and that it contains the table
	sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_key=%s", filePath, "key"))
	if err != nil {
		t.Fatalf("Could not open database: %v", err)
	}

	defer sqlDB.Close()

	results := sqlDB.QueryRow(`SELECT count(*) FROM sqlite_master
		WHERE type='table' AND name in ("clusters", "users", "checkerResults", "dismissals");`)

	var count int
	if err := results.Scan(&count); err != nil {
		t.Fatalf("could not scan the database: %v", err)
	}

	if count != 4 {
		t.Fatalf("The number of tables is incorrect, expected %d got %d", 4, count)
	}
}

func TestNewSQLiteDBExistButInvalid(t *testing.T) {
	testDir := t.TempDir()

	filePath := filepath.Join(testDir, "db.sqlite")

	// verify that the file was created and that it contains the table
	sqlDB, err := sql.Open("sqlite3", fmt.Sprintf("%s?_key=%s", filePath, "key"))
	if err != nil {
		t.Fatalf("Could not open database: %v", err)
	}

	_, err = sqlDB.Exec(fmt.Sprintf("PRAGMA key = '%s';", "key"))
	if err != nil {
		t.Fatalf("could not set key: %v", err)
	}

	// create cluster table
	_, err = sqlDB.Exec(`
		CREATE TABLE clusters (
			uuid VARCHAR(50) NOT NULL PRIMARY KEY,
			name VARCHAR(300),
			nodes BLOB NOT NULL,
			buckets BLOB,
			info BLOB,
			user  VARCHAR(300) NOT NULL,
			password VARCHAR(300) NOT NULL,
			heartbeatIssue INT DEFAULT 0,
			lastUpdate BLOB,
			cacert BLOB
		);`)
	if err != nil {
		t.Fatalf("Could not create table for test: %v", err)
	}

	_ = sqlDB.Close()

	_, err = NewSQLiteDB(filePath, "key")
	if err == nil {
		t.Fatalf("Expected an error but got <nil>")
	}
}

func TestDBIsInitialized(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	init, err := db.IsInitialized()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if init {
		t.Fatal("Expected initialized to be false")
	}

	err = db.AddUser(&values.User{
		User:     "a",
		Password: []byte(`alpha`),
		Admin:    true,
	})
	if err != nil {
		t.Fatalf("Could not add user: %v", err)
	}

	init, err = db.IsInitialized()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !init {
		t.Fatal("Expected initialized to be true")
	}
}

func TestDBUpgrades(t *testing.T) {
	db, _ := createEmptyDBOnVersion0(t)
	for targetVersion := 1; targetVersion <= CurrentVersion; targetVersion++ {
		t.Run(fmt.Sprintf("%d-to-%d", targetVersion-1, targetVersion), func(t *testing.T) {
			for runUpgradeVersion := 1; runUpgradeVersion < targetVersion; runUpgradeVersion++ {
				err := storeUpgradeFunctions[Version(runUpgradeVersion)](db.sqlDB)
				require.NoErrorf(t, err, "failed upgrade to %d", runUpgradeVersion)
				row := db.sqlDB.QueryRow("PRAGMA user_version")

				// Verify that the upgrade actually set the user_version
				var userVersion Version
				err = row.Scan(&userVersion)
				require.NoError(t, err)
				require.Equal(t, runUpgradeVersion, int(userVersion), "upgrade did not set user_version")
			}
		})
	}
}
