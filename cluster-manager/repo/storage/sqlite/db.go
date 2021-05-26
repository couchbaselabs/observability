package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/couchbaselabs/cbmultimanager/storage"

	// sqlcipher driver import
	_ "github.com/xeodou/go-sqlcipher"
)

type Version uint8

const VersionOne = 1

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
		err = setupNewSQLDB(SQLDB, key)
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

	if userVersion != VersionOne {
		return fmt.Errorf("unknown sqlite DB version %d", VersionOne)
	}

	// confirm that the tables we need exists
	results := db.sqlDB.QueryRow(`
		SELECT count(*) FROM sqlite_master
		WHERE type='table' AND name in ("clusters", "users", "checkerResults", "dismissals");`)

	var count int
	if err := results.Scan(&count); err != nil {
		return fmt.Errorf("could not confimr if tables exist: %w", err)
	}

	if count != 4 {
		return fmt.Errorf("the required tables do not exist")
	}
	// TODO verify schemas

	return nil
}

func setupNewSQLDB(db *DB, key string) error {
	// setup the key
	_, err := db.sqlDB.Exec(fmt.Sprintf("PRAGMA key = '%s';", key))
	if err != nil {
		return fmt.Errorf("could not setup key: %w", err)
	}

	_, err = db.sqlDB.Exec(fmt.Sprintf("PRAGMA user_version= %d", VersionOne))
	if err != nil {
		return fmt.Errorf("could not set user_version: %w", err)
	}

	// create cluster table
	_, err = db.sqlDB.Exec(`
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
		return fmt.Errorf("could not create clusters table: %w", err)
	}

	// create user table for cluster manager users
	_, err = db.sqlDB.Exec(`
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
	_, err = db.sqlDB.Exec(`
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
	_, err = db.sqlDB.Exec(`
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
