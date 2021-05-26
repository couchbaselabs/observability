package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/values"

	_ "github.com/xeodou/go-sqlcipher"
)

func createEmptyDB(t *testing.T) (storage.Store, string) {
	testDir := t.TempDir()

	filePath := filepath.Join(testDir, "db.sqlite")
	db, err := NewSQLiteDB(filePath, "key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
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
