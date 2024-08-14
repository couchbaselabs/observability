// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (db *DB) AddCloudCredentials(creds *values.Credential) error {
	if creds.AccessKey == "" || creds.SecretKey == "" || creds.Name == "" {
		return fmt.Errorf("access key, secret key and name are required")
	}

	creds.DateAdded = time.Now().UTC()

	_, err := db.sqlDB.Exec(`INSERT INTO cloudCreds (name, accessKey, secretKey, dateAdded) VALUES (?, ?, ?, ?);`,
		creds.Name, creds.AccessKey, creds.SecretKey, creds.DateAdded)
	if err != nil {
		return fmt.Errorf("could not add credentials: %w", err)
	}

	return nil
}

func (db *DB) GetCloudCredentials(sensitive bool) ([]*values.Credential, error) {
	querySelect := "name, dateAdded"
	if sensitive {
		querySelect += ", accessKey, secretKey"
	}

	rows, err := db.sqlDB.Query(`SELECT ` + querySelect + ` FROM cloudCreds;`)
	if err != nil {
		return nil, fmt.Errorf("could not get credentials: %w", err)
	}
	defer rows.Close()

	var creds []*values.Credential
	if sensitive {
		creds, err = sensitiveCredsScan(rows)
	} else {
		creds, err = credScan(rows)
	}

	if err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return creds, nil
}

func sensitiveCredsScan(rows *sql.Rows) ([]*values.Credential, error) {
	var creds []*values.Credential
	for rows.Next() {
		var cred values.Credential
		if err := rows.Scan(&cred.Name, &cred.DateAdded, &cred.AccessKey, &cred.SecretKey); err != nil {
			return nil, fmt.Errorf("could not scan credentials: %w", err)
		}

		creds = append(creds, &cred)
	}

	return creds, nil
}

func credScan(rows *sql.Rows) ([]*values.Credential, error) {
	var creds []*values.Credential
	for rows.Next() {
		var cred values.Credential
		if err := rows.Scan(&cred.Name, &cred.DateAdded); err != nil {
			return nil, fmt.Errorf("could not scan credentials: %w", err)
		}

		creds = append(creds, &cred)
	}

	return creds, nil
}
