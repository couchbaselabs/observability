// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (db *DB) SetCheckerResult(result *values.WrappedCheckerResult) error {
	if result.Error != nil {
		return fmt.Errorf("cannot store results with errors")
	}

	if result.Result == nil {
		return fmt.Errorf("checker result is required")
	}

	timeByte, err := json.Marshal(result.Result.Time)
	if err != nil {
		return fmt.Errorf("could not marshal result time: %w", err)
	}

	// the query below should ensure we only keep the latest results
	_, err = db.sqlDB.Exec(`
		REPLACE INTO checkerResults
			(name, remediation, value, status, time, version, clusterUUID, nodeUUID, logFile, bucketName)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		result.Result.Name, result.Result.Remediation, result.Result.Value, result.Result.Status, timeByte,
		result.Result.Version, result.Cluster, result.Node, result.LogFile, result.Bucket)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	return nil
}

func (db *DB) DeleteCheckerResults(search values.CheckerSearch) error {
	// build the where clause based on the given parameters
	whereClauseParts := make([]string, 0)
	whereClauseTerms := make([]interface{}, 0)

	if search.Name != nil {
		whereClauseParts = append(whereClauseParts, "name = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Name)
	}

	if search.Cluster != nil {
		whereClauseParts = append(whereClauseParts, "clusterUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Cluster)
	}

	if search.Node != nil {
		whereClauseParts = append(whereClauseParts, "nodeUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Node)
	}

	if search.LogFile != nil {
		whereClauseParts = append(whereClauseParts, "logFile = ?")
		whereClauseTerms = append(whereClauseTerms, *search.LogFile)
	}

	if search.Bucket != nil {
		whereClauseParts = append(whereClauseParts, "bucketName = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Bucket)
	}

	if len(whereClauseParts) == 0 {
		return fmt.Errorf("a search term is required to delete")
	}

	_, err := db.sqlDB.Exec("DELETE FROM checkerResults WHERE "+strings.Join(whereClauseParts, " AND "),
		whereClauseTerms...)
	if err != nil {
		return fmt.Errorf("could not delete results: %w", err)
	}

	return nil
}

func (db *DB) GetCheckerResult(search values.CheckerSearch) ([]*values.WrappedCheckerResult, error) {
	// build the where clause based on the given parameters
	whereClauseParts := make([]string, 0)
	whereClauseTerms := make([]interface{}, 0)

	if search.Name != nil {
		whereClauseParts = append(whereClauseParts, "name = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Name)
	}

	if search.Cluster != nil {
		whereClauseParts = append(whereClauseParts, "clusterUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Cluster)
	}

	if search.Node != nil {
		whereClauseParts = append(whereClauseParts, "nodeUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Node)
	}

	if search.LogFile != nil {
		whereClauseParts = append(whereClauseParts, "logFile = ?")
		whereClauseTerms = append(whereClauseTerms, *search.LogFile)
	}

	if search.Bucket != nil {
		whereClauseParts = append(whereClauseParts, "bucketName = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Bucket)
	}

	var whereClause string
	if len(whereClauseParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauseParts, " AND ")
	}

	rows, err := db.sqlDB.Query(`
		SELECT
			name, remediation, value, status, time, version, clusterUUID, nodeUUID, logFile, bucketName
		FROM checkerResults`+whereClause+" ORDER BY name, clusterUUID, nodeUUID, bucketName, logFile;",
		whereClauseTerms...)
	if err != nil {
		return nil, fmt.Errorf("could not perform select: %w", err)
	}

	defer rows.Close()

	results := make([]*values.WrappedCheckerResult, 0)
	for rows.Next() {
		res := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{},
		}

		var byteTime, value []byte

		err = rows.Scan(&res.Result.Name, &res.Result.Remediation, &value, &res.Result.Status, &byteTime,
			&res.Result.Version, &res.Cluster, &res.Node, &res.LogFile, &res.Bucket)
		if err != nil {
			return nil, fmt.Errorf("could not scan value: %w", err)
		}

		if err = json.Unmarshal(byteTime, &res.Result.Time); err != nil {
			return nil, fmt.Errorf("could not unmarshal result time: %w", err)
		}

		if len(value) != 0 {
			if err = json.Unmarshal(value, &res.Result.Value); err != nil {
				return nil, fmt.Errorf("could not unmarshal value: %w", err)
			}
		}

		results = append(results, res)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return results, nil
}

func (db *DB) DeleteWhereNodesDoNotMatch(knownNodes []string, clusterUUID string) (int64, error) {
	if len(knownNodes) == 0 {
		return 0, fmt.Errorf("at least one node is needed")
	}

	quoted := make([]string, len(knownNodes))
	for i, node := range knownNodes {
		quoted[i] = `"` + node + `"`
	}

	result, err := db.sqlDB.Exec(fmt.Sprintf(`
		DELETE FROM checkerResults
		WHERE clusterUUID = ? AND nodeUUID IS NOT NULL AND nodeUUID != "" AND nodeUUID NOT IN (%s);`,
		strings.Join(quoted, ", ")), clusterUUID)
	if err != nil {
		return 0, fmt.Errorf("could not delete results: %w", err)
	}

	return result.RowsAffected()
}

func (db *DB) DeleteWhereBucketsDoNotMatch(knownBuckets []string, clusterUUID string) (int64, error) {
	if len(knownBuckets) == 0 {
		return 0, fmt.Errorf("at least one bucket is needed")
	}

	quoted := make([]string, len(knownBuckets))
	for i, bucket := range knownBuckets {
		quoted[i] = `"` + bucket + `"`
	}

	result, err := db.sqlDB.Exec(fmt.Sprintf(`
		DELETE FROM checkerResults
		WHERE clusterUUID = ? AND bucketName IS NOT NULL AND bucketName != "" AND bucketName NOT IN (%s);`,
		strings.Join(quoted, ", ")), clusterUUID)
	if err != nil {
		return 0, fmt.Errorf("could not delete results: %w", err)
	}

	return result.RowsAffected()
}
