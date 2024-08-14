// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"go.uber.org/zap"
)

func (db *DB) AddCluster(cluster *values.CouchbaseCluster) error {
	if len(cluster.NodesSummary) == 0 {
		return fmt.Errorf("hosts are required")
	}

	nodes, err := json.Marshal(cluster.NodesSummary)
	if err != nil {
		return fmt.Errorf("invalid nodes summary: %w", err)
	}

	// to deal with nil values
	if cluster.CaCert == nil {
		cluster.CaCert = []byte{}
	}

	var buckets []byte
	if cluster.BucketsSummary != nil {
		buckets, err = json.Marshal(cluster.BucketsSummary)
		if err != nil {
			return fmt.Errorf("could not marshall buckets summary: %w", err)
		}
	}

	var remoteClusters []byte
	if cluster.RemoteClusters != nil {
		remoteClusters, err = json.Marshal(cluster.RemoteClusters)
		if err != nil {
			return fmt.Errorf("could not marshall remtoe clusters: %w", err)
		}
	}

	var info []byte
	if cluster.ClusterInfo != nil {
		info, err = json.Marshal(cluster.ClusterInfo)
		if err != nil {
			return fmt.Errorf("could not marshall cluster info: %w", err)
		}
	}

	byteTime, err := json.Marshal(time.Now())
	if err != nil {
		return fmt.Errorf("could not marshal current time: %w", err)
	}

	tx, err := db.sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO clusters (uuid, enterprise, name, nodes, buckets, remoteClusters, 
			info,  user, password, heartbeatIssue, cacert, lastUpdate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`, cluster.UUID, cluster.Enterprise,
		cluster.Name, nodes, buckets, remoteClusters, info, cluster.User, cluster.Password,
		cluster.HeartBeatIssue, cluster.CaCert, byteTime)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("could not add cluster: %w", err)
	}

	if cluster.Alias == "" {
		return tx.Commit()
	}

	_, err = tx.Exec("INSERT INTO aliases (alias, clusterUUID) VALUES (?, ?);", cluster.Alias, cluster.UUID)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			zap.S().Errorw("(SQLCIPHER) Could not rollback alias transaction cluster may have been added", "err", err)
		}

		return fmt.Errorf("could not add alias: %w", err)
	}

	return tx.Commit()
}

func (db *DB) GetClusters(sensitive bool, enterpriseOnly bool) ([]*values.CouchbaseCluster, error) {
	clusters := make([]*values.CouchbaseCluster, 0)

	parameters := "uuid, enterprise, name, nodes, buckets, remoteClusters, info, heartbeatIssue, lastUpdate, alias"
	if sensitive {
		parameters += ", user, password, cacert"
	}

	var where string
	if enterpriseOnly {
		where = "WHERE enterprise = true "
	}

	rows, err := db.sqlDB.Query("SELECT " + parameters +
		" FROM clusters LEFT JOIN aliases on aliases.clusterUUID = clusters.uuid " + where + "ORDER BY uuid ASC;")
	if err != nil {
		return nil, fmt.Errorf("could not get clusters: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		cluster, err := scanCluster(rows, sensitive)
		if err != nil {
			return nil, fmt.Errorf("failed scanning cluster: %w", err)
		}

		clusters = append(clusters, cluster)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through rows: %w", err)
	}

	return clusters, nil
}

func (db *DB) GetCluster(uuid string, sensitive bool) (*values.CouchbaseCluster, error) {
	parameters := "uuid, enterprise, name, nodes, buckets, remoteClusters, info, heartbeatIssue, lastUpdate, alias"
	if sensitive {
		parameters += ", user, password, cacert"
	}

	row := db.sqlDB.QueryRow("SELECT "+parameters+
		" FROM clusters LEFT JOIN aliases ON aliases.clusterUUID = clusters.uuid WHERE uuid = ?;", uuid)

	cluster, err := scanCluster(row, sensitive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, values.ErrNotFound
		}

		return nil, fmt.Errorf("failed scanning cluster: %w", err)
	}

	return cluster, nil
}

func (db *DB) DeleteCluster(uuid string) error {
	_, err := db.sqlDB.Exec("DELETE FROM clusters WHERE uuid = ?;", uuid)
	if err != nil {
		return fmt.Errorf("could not delete cluster: %w", err)
	}

	return nil
}

// UpdateCluster will update the value of any field in the cluster that is not empty/null. The uuid is immutable so that
// does not change.
func (db *DB) UpdateCluster(cluster *values.CouchbaseCluster) error {
	byteTime, err := json.Marshal(time.Now())
	if err != nil {
		return fmt.Errorf("could not marshal current time: %w", err)
	}

	parameters := []string{"heartbeatIssue = ?", "lastUpdate = ?"}
	toScan := []interface{}{cluster.HeartBeatIssue, byteTime}

	// In any of the other cases we could not establish a connection with the other cluster so the enterprise value is
	// not reliable. If it is a UUIDMismatch issue we have to update to avoid a potential circumvent of CE clusters
	// getting health checks.
	if cluster.HeartBeatIssue == values.NoHeartIssue || cluster.HeartBeatIssue == values.UUIDMismatchHeartIssue {
		parameters = append(parameters, "enterprise = ?")
		toScan = append(toScan, cluster.Enterprise)
	}

	if cluster.Name != "" {
		parameters = append(parameters, "name = ?")
		toScan = append(toScan, cluster.Name)
	}

	if cluster.User != "" {
		parameters = append(parameters, "user = ?")
		toScan = append(toScan, cluster.User)
	}

	if cluster.Password != "" {
		parameters = append(parameters, "password = ?")
		toScan = append(toScan, cluster.Password)
	}

	if cluster.NodesSummary != nil {
		hosts, err := json.Marshal(cluster.NodesSummary)
		if err != nil {
			return fmt.Errorf("could not marshal nodes summary: %w", err)
		}

		parameters = append(parameters, "nodes = ?")
		toScan = append(toScan, hosts)
	}

	if cluster.RemoteClusters != nil {
		remoteClusters, err := json.Marshal(cluster.RemoteClusters)
		if err != nil {
			return fmt.Errorf("could not marshall remote clusters: %w", err)
		}

		parameters = append(parameters, "remoteClusters = ?")
		toScan = append(toScan, remoteClusters)
	}

	if cluster.BucketsSummary != nil {
		buckets, err := json.Marshal(cluster.BucketsSummary)
		if err != nil {
			return fmt.Errorf("could not marshall bucket summary: %w", err)
		}

		parameters = append(parameters, "buckets = ?")
		toScan = append(toScan, buckets)
	}

	if cluster.ClusterInfo != nil {
		info, err := json.Marshal(cluster.ClusterInfo)
		if err != nil {
			return fmt.Errorf("could not marashll cluster info: %w", err)
		}

		parameters = append(parameters, "info = ?")
		toScan = append(toScan, info)
	}

	if cluster.CaCert != nil {
		parameters = append(parameters, "cacert = ?")
		toScan = append(toScan, cluster.CaCert)
	}

	toScan = append(toScan, cluster.UUID)

	_, err = db.sqlDB.Exec("UPDATE clusters SET "+strings.Join(parameters, ", ")+" WHERE uuid = ?;", toScan...)
	if err != nil {
		return fmt.Errorf("could not update cluster: %w", err)
	}

	return nil
}

func scanCluster(row scannable, sensitive bool) (*values.CouchbaseCluster, error) {
	var cluster values.CouchbaseCluster
	var nodes, byteTime, buckets, remoteClusters, info, alias []byte

	var err error
	if sensitive {
		err = row.Scan(&cluster.UUID, &cluster.Enterprise, &cluster.Name, &nodes, &buckets, &remoteClusters, &info,
			&cluster.HeartBeatIssue, &byteTime, &alias, &cluster.User, &cluster.Password, &cluster.CaCert)
	} else {
		err = row.Scan(&cluster.UUID, &cluster.Enterprise, &cluster.Name, &nodes, &buckets, &remoteClusters, &info,
			&cluster.HeartBeatIssue, &byteTime, &alias)
	}

	if err != nil {
		return nil, err
	}

	if len(alias) > 0 {
		cluster.Alias = string(alias)
	}

	if err = json.Unmarshal(nodes, &cluster.NodesSummary); err != nil {
		return nil, fmt.Errorf("could not unmarshal nodes for cluster '%s': %w", cluster.UUID, err)
	}

	if err = json.Unmarshal(byteTime, &cluster.LastUpdate); err != nil {
		return nil, fmt.Errorf("could not unmarshal last update time for cluster '%s': %w", cluster.UUID, err)
	}

	if len(buckets) != 0 {
		if err = json.Unmarshal(buckets, &cluster.BucketsSummary); err != nil {
			return nil, fmt.Errorf("could not unmarshal buckets for cluster '%s': %w", cluster.UUID, err)
		}
	}

	if len(remoteClusters) != 0 {
		if err = json.Unmarshal(remoteClusters, &cluster.RemoteClusters); err != nil {
			return nil, fmt.Errorf("could not unmarshal remote clusters for cluster '%s': %w", cluster.UUID, err)
		}
	}

	if len(info) != 0 {
		if err = json.Unmarshal(info, &cluster.ClusterInfo); err != nil {
			return nil, fmt.Errorf("could not unmarshal info for cluster '%s': %w", cluster.UUID, err)
		}
	}

	return &cluster, nil
}
