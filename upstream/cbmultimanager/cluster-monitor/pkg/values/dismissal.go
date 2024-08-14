// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import "time"

// DismissLevel controls the level at which the dismissal works.
type DismissLevel uint8

const (
	AllDismissLevel DismissLevel = iota
	ClusterDismissLevel
	BucketDismissLevel
	NodeDismissLevel
	FileDismissLevel
)

// Dismissal contains information of for how long a checker is dismissed for and what granularity the checker
// dismissal has.
type Dismissal struct {
	// handler until when somethings is dismissed
	Forever bool      `json:"forever,omitempty"`
	Until   time.Time `json:"until,omitempty"`

	Level       DismissLevel `json:"level"`
	ClusterUUID string       `json:"cluster_uuid,omitempty"`
	BucketName  string       `json:"bucket_name,omitempty"`
	LogFile     string       `json:"log_file,omitempty"`
	NodeUUID    string       `json:"node_uuid,omitempty"`

	ID          string `json:"id"`
	CheckerName string `json:"checker_name"`
}

// IsDismissed will check if the dismissal applies to the given result and if it is still valid. It will return a
// boolean denoting if the result should be dismissed or not.
func (d *Dismissal) IsDismissed(result *WrappedCheckerResult) bool {
	if result.Result.Name != d.CheckerName {
		return false
	}

	if !d.Forever && time.Now().After(d.Until) {
		return false
	}

	switch d.Level {
	case AllDismissLevel:
		return true
	case ClusterDismissLevel:
		return result.Cluster == d.ClusterUUID
	case BucketDismissLevel:
		return result.Cluster == d.ClusterUUID && result.Bucket == d.BucketName
	case NodeDismissLevel:
		return result.Cluster == d.ClusterUUID && result.Node == d.NodeUUID
	case FileDismissLevel:
		// TODO this may need to change to regex. I'll think about it once I do the log file ones
		return result.Cluster == d.ClusterUUID && result.Node == d.NodeUUID && result.LogFile == d.LogFile
	default:
		return false
	}
}

// DismissalSearchSpace is used to filter what to get/delete/update in the dismissal store.
type DismissalSearchSpace struct {
	ID          *string
	CheckerName *string
	ClusterUUID *string
	BucketName  *string
	NodeUUID    *string
	LogFile     *string
	Level       *DismissLevel
}
