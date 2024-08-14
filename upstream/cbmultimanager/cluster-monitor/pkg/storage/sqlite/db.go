// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	// sqlcipher driver import
	_ "github.com/xeodou/go-sqlcipher"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
)

type Version uint8

// CurrentVersion is the latest user_version of the SQLite store that this cbmultimanager supports.
// On startup, any store with a user_version below the CurrentVersion will be upgraded to it.
const CurrentVersion = 2

// storeUpgradeFunctions has the functions to upgrade the DB from an older version. In general,
// storeUpgradeFunctions[N] must execute the SQL needed to upgrade the DB from version N-1 to N, including incrementing
// the user_version.
var storeUpgradeFunctions = map[Version]func(db *sql.DB) error{
	1: func(db *sql.DB) error {
		// create cluster table
		_, err := db.Exec(`
		CREATE TABLE clusters (
			uuid VARCHAR(50) NOT NULL PRIMARY KEY,
			enterprise BOOLEAN,
			name VARCHAR(300),
			nodes BLOB NOT NULL,
			buckets BLOB,
			remoteClusters BLOB,
			info BLOB,
			user  VARCHAR(300) NOT NULL,
			password VARCHAR(300) NOT NULL,
			heartbeatIssue INT DEFAULT 0,
			lastUpdate BLOB,
			cacert BLOB
		);`)
		if err != nil {
			return fmt.Errorf("could not create clusters table: %w", err)
		}

		// create user table for cluster manager users
		_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER NOT NULL PRIMARY KEY,
			user VARCHAR(256) NOT NULL UNIQUE,
			password BLOB NOT NULL,
			admin BOOLEAN
		);`)
		if err != nil {
			return fmt.Errorf("could not create users table: %w", err)
		}

		// create user table for cluster manager users
		_, err = db.Exec(`
		CREATE TABLE checkerResults (
			name VARCHAR(300) NOT NULL,
			remediation TEXT,
			value BLOB,
			status VARCHAR(50),
			time  BLOB,
			version INT DEFAULT 0,
			clusterUUID VARCHAR(100) NOT NULL,
			nodeUUID VARCHAR(100) NOT NULL,
			bucketName VARCHAR(300) NOT NULL,
			logFile TEXT NOT NULL,
			PRIMARY KEY (name, clusterUUID, nodeUUID, logFile, bucketName)
		);`)
		if err != nil {
			return fmt.Errorf("could not create checker result table: %w", err)
		}

		// create a table that deals with dismissing checkers
		_, err = db.Exec(`
		CREATE TABLE dismissals (
			id VARCHAR(300) NOT NULL PRIMARY KEY,
			level INT NOT NULL,
			checkerName VARCHAR(300) NOT NULL,
			clusterUUID VARCHAR(50),
			bucket VARCHAR(300),
			nodeUUID VARCHAR(300),
			file VARCHAR(300),
			forever BOOLEAN NOT NULL,
			until TIMESTAMP
		);`)
		if err != nil {
			return fmt.Errorf("could not create dismissal table: %w", err)
		}

		// create a table for aliases
		_, err = db.Exec(`
		CREATE TABLE aliases (
		    alias VARCHAR(300) NOT NULL PRIMARY KEY,
		    clusterUUID VARCHAR(50) NOT NULL UNIQUE REFERENCES clusters(uuid) ON DELETE CASCADE
		);`)
		if err != nil {
			return fmt.Errorf("could not create dismissal table: %w", err)
		}

		_, err = db.Exec(`
		CREATE TABLE cloudCreds (
		    name VARCHAR (300) NOT NULL PRIMARY KEY,
		    accessKey NOT NULL,
		    secretKey NOT NULL,
		    dateAdded TIMESTAMP
		)`)
		if err != nil {
			return fmt.Errorf("could not create cloud credentials table: %w", err)
		}

		_, err = db.Exec("PRAGMA user_version=1;")
		if err != nil {
			return fmt.Errorf("could not set user_version: %w", err)
		}
		return nil
	},
	2: func(db *sql.DB) error {
		_, err := db.Exec(`
			ALTER TABLE checkerResults
			RENAME TO checkerResults_tmp
		`)
		if err != nil {
			return fmt.Errorf("could not execute first rename: %w", err)
		}
		_, err = db.Exec(`
			CREATE TABLE checkerResults (
			name VARCHAR(300) NOT NULL,
			remediation TEXT,
			value BLOB,
			status VARCHAR(50),
			time  BLOB,
			version INT DEFAULT 0,
			clusterUUID VARCHAR(100) NOT NULL REFERENCES clusters(uuid),
			nodeUUID VARCHAR(100) NOT NULL,
			bucketName VARCHAR(300) NOT NULL,
			logFile TEXT NOT NULL,
			PRIMARY KEY (name, clusterUUID, nodeUUID, logFile, bucketName)
		)
		`)
		if err != nil {
			return fmt.Errorf("could not execute CREATE TABLE: %w", err)
		}
		_, err = db.Exec(`
			INSERT INTO checkerResults
			SELECT * FROM checkerResults_tmp
		`)
		if err != nil {
			return fmt.Errorf("failed to copy data to new table: %w", err)
		}
		_, err = db.Exec(`DROP TABLE checkerResults_tmp`)
		if err != nil {
			return fmt.Errorf("failed to drop temporary table: %w", err)
		}
		_, err = db.Exec("PRAGMA foreign_keys=ON")
		if err != nil {
			return fmt.Errorf("could not enable foreign keys: %w", err)
		}
		_, err = db.Exec("PRAGMA user_version=2;")
		if err != nil {
			return fmt.Errorf("could not set user_version: %w", err)
		}
		return nil
	},
}

type scannable interface {
	Scan(dest ...interface{}) error
}

type DB struct {
	fileName string
	sqlDB    *sql.DB
}

func NewSQLiteDB(fileName, key string) (storage.Store, error) {
	_, err := os.Stat(fileName)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not confirm SQLite DB exists")
	}

	exists := !os.IsNotExist(err)

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_key=%s", fileName, key))
	if err != nil {
		return nil, fmt.Errorf("could not open sqlite database: %w", err)
	}

	SQLDB := &DB{
		fileName: fileName,
		sqlDB:    db,
	}

	if exists {
		err = verifyStore(SQLDB)
	} else {
		err = setupNewSQLDB(SQLDB, key, true)
	}

	if err != nil {
		return nil, fmt.Errorf("error initializing store: %w", err)
	}

	return SQLDB, nil
}

func verifyStore(db *DB) error {
	row := db.sqlDB.QueryRow("PRAGMA user_version")
	var userVersion Version
	if err := row.Scan(&userVersion); err != nil {
		return fmt.Errorf("could not get user version")
	}

	switch {
	case userVersion == CurrentVersion:
		// nothing to do
	case userVersion < CurrentVersion:
		if err := upgradeStoreToLatest(db, userVersion); err != nil {
			return fmt.Errorf("failed to upgrade store from version %d to %d: %w", userVersion, CurrentVersion, err)
		}
	case userVersion > CurrentVersion:
		return fmt.Errorf("cbmultimanager is outdated: store schema version %d is newer than our version (%d)",
			userVersion, CurrentVersion)
	}

	// confirm that the tables we need exists
	// the interface{} is because that's the parameter type of QueryRow
	requiredTables := []interface{}{"clusters", "users", "checkerResults", "dismissals", "aliases"}
	requiredTableParams := strings.TrimSuffix(strings.Repeat("?,", len(requiredTables)), ",")
	results := db.sqlDB.QueryRow(fmt.Sprintf(`
		SELECT count(*) FROM sqlite_master
		WHERE type='table' AND name in (%s);`, requiredTableParams),
		requiredTables...)

	var count int
	if err := results.Scan(&count); err != nil {
		return fmt.Errorf("could not confirm if tables exist: %w", err)
	}

	if count != len(requiredTables) {
		return fmt.Errorf("possibly corrupt store: the required tables do not exist")
	}
	// TODO verify schemas

	return nil
}

func upgradeStoreToLatest(db *DB, initialVersion Version) error {
	for target := initialVersion + 1; target <= CurrentVersion; target++ {
		upgrade, ok := storeUpgradeFunctions[target]
		if !ok {
			return fmt.Errorf("no upgrade function for version %d", target)
		}
		if err := upgrade(db.sqlDB); err != nil {
			return fmt.Errorf("failed to upgrade to version %d: %w", target, err)
		}
		zap.S().Infow("(Store / SQLite) Upgraded store", "newVersion", target)
	}
	return nil
}

func setupNewSQLDB(db *DB, key string, shouldUpgrade bool) error {
	// setup the key
	_, err := db.sqlDB.Exec(fmt.Sprintf("PRAGMA key = '%s';", key))
	if err != nil {
		return fmt.Errorf("could not setup key: %w", err)
	}

	_, err = db.sqlDB.Exec(fmt.Sprintf("PRAGMA user_version= %d", 1))
	if err != nil {
		return fmt.Errorf("could not set user_version: %w", err)
	}

	_, err = db.sqlDB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return fmt.Errorf("could not set foreign keys on: %w", err)
	}

	if shouldUpgrade {
		err = upgradeStoreToLatest(db, 0)
		if err != nil {
			return fmt.Errorf("could not upgrade new store: %w", err)
		}
	}

	return nil
}

// IsInitialized returns true if there is at least one admin user false otherwise
func (db *DB) IsInitialized() (bool, error) {
	result := db.sqlDB.QueryRow(`SELECT count(id) FROM users WHERE admin=1;`)
	var count int
	if err := result.Scan(&count); err != nil {
		return false, fmt.Errorf("could not check if store initialized: %w", err)
	}

	return count > 0, nil
}

func (db *DB) Close() error {
	return db.sqlDB.Close()
}
