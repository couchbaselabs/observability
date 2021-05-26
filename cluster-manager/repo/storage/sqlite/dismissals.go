package sqlite

import (
	"fmt"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

func (db *DB) AddDismissal(dismissal values.Dismissal) error {
	if dismissal.ID == "" {
		return fmt.Errorf("id is required")
	}

	_, err := db.sqlDB.Exec(`
		INSERT INTO dismissals (id, level, checkerName, clusterUUID, bucket, nodeUUID, file, forever, until)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`, dismissal.ID, dismissal.Level, dismissal.CheckerName,
		dismissal.ClusterUUID, dismissal.BucketName, dismissal.NodeUUID, dismissal.LogFile, dismissal.Forever,
		dismissal.Until.UTC())
	if err != nil {
		return fmt.Errorf("could not insert dismissal: %w", err)
	}

	return nil
}

func (db *DB) GetDismissals(search values.DismissalSearchSpace) ([]*values.Dismissal, error) {
	whereClauseParts, whereClauseTerms := buildWhereClause(search)

	var whereClause string
	if len(whereClauseTerms) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauseParts, " AND ")
	}

	rows, err := db.sqlDB.Query(`
		SELECT id, level, checkerName, clusterUUID, bucket, nodeUUID, file, forever, until
		FROM dismissals`+whereClause+" ORDER BY id;", whereClauseTerms...)
	if err != nil {
		return nil, fmt.Errorf("could not get requested dismissals: %w", err)
	}

	defer rows.Close()

	dismissals := make([]*values.Dismissal, 0)
	for rows.Next() {
		var dismissal values.Dismissal
		err = rows.Scan(&dismissal.ID, &dismissal.Level, &dismissal.CheckerName, &dismissal.ClusterUUID,
			&dismissal.BucketName, &dismissal.NodeUUID, &dismissal.LogFile, &dismissal.Forever, &dismissal.Until)
		if err != nil {
			return nil, fmt.Errorf("error scanning dismissal row: %w", err)
		}

		dismissals = append(dismissals, &dismissal)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return dismissals, nil
}

func (db *DB) DeleteDismissals(search values.DismissalSearchSpace) error {
	whereClauseParts, whereClauseTerms := buildWhereClause(search)

	if len(whereClauseParts) == 0 {
		return fmt.Errorf("at least one search term is required")
	}

	_, err := db.sqlDB.Exec("DELETE FROM dismissals WHERE "+strings.Join(whereClauseParts, " AND "), whereClauseTerms...)
	if err != nil {
		return fmt.Errorf("could not delete dismissals: %w", err)
	}

	return nil
}

func (db *DB) DeleteExpiredDismissals() (int64, error) {
	result, err := db.sqlDB.Exec("DELETE FROM dismissals WHERE until IS NOT NULL AND forever == 0 AND until < ?;",
		time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("could not delete expired dismissals: %w", err)
	}

	return result.RowsAffected()
}

func (db *DB) DeleteDismissalForUnknownBuckets(knownBuckets []string, clusterUUID string) (int64, error) {
	if len(knownBuckets) == 0 {
		return 0, fmt.Errorf("at least one bucket is needed")
	}

	quoted := make([]string, len(knownBuckets))
	for i, bucket := range knownBuckets {
		quoted[i] = `"` + bucket + `"`
	}

	result, err := db.sqlDB.Exec(fmt.Sprintf(`
		DELETE FROM dismissals
		WHERE clusterUUID = ? AND bucket IS NOT NULL AND bucket != "" AND bucket NOT IN (%s);`,
		strings.Join(quoted, ", ")), clusterUUID)
	if err != nil {
		return 0, fmt.Errorf("could not delete results: %w", err)
	}

	return result.RowsAffected()
}

func (db *DB) DeleteDismissalForUnknownNodes(knownNodes []string, clusterUUID string) (int64, error) {
	if len(knownNodes) == 0 {
		return 0, fmt.Errorf("at least one node is needed")
	}

	quoted := make([]string, len(knownNodes))
	for i, node := range knownNodes {
		quoted[i] = `"` + node + `"`
	}

	result, err := db.sqlDB.Exec(fmt.Sprintf(`
		DELETE FROM dismissals
		WHERE clusterUUID = ? AND nodeUUID IS NOT NULL AND nodeUUID != "" AND nodeUUID NOT IN (%s);`,
		strings.Join(quoted, ", ")), clusterUUID)
	if err != nil {
		return 0, fmt.Errorf("could not delete results: %w", err)
	}

	return result.RowsAffected()
}

func buildWhereClause(search values.DismissalSearchSpace) ([]string, []interface{}) {
	// build the where clause based on the given parameters
	whereClauseParts := make([]string, 0)
	whereClauseTerms := make([]interface{}, 0)

	if search.ID != nil {
		whereClauseParts = append(whereClauseParts, "id = ?")
		whereClauseTerms = append(whereClauseTerms, *search.ID)
	}

	if search.CheckerName != nil {
		whereClauseParts = append(whereClauseParts, "checkerName = ?")
		whereClauseTerms = append(whereClauseTerms, *search.CheckerName)
	}

	if search.ClusterUUID != nil {
		whereClauseParts = append(whereClauseParts, "clusterUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.ClusterUUID)
	}

	if search.BucketName != nil {
		whereClauseParts = append(whereClauseParts, "bucket = ?")
		whereClauseTerms = append(whereClauseTerms, *search.BucketName)
	}

	if search.NodeUUID != nil {
		whereClauseParts = append(whereClauseParts, "nodeUUID = ?")
		whereClauseTerms = append(whereClauseTerms, *search.NodeUUID)
	}

	if search.LogFile != nil {
		whereClauseParts = append(whereClauseParts, "file = ?")
		whereClauseTerms = append(whereClauseTerms, *search.LogFile)
	}

	if search.Level != nil {
		whereClauseParts = append(whereClauseParts, "level = ?")
		whereClauseTerms = append(whereClauseTerms, *search.Level)
	}

	return whereClauseParts, whereClauseTerms
}
