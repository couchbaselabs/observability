// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

type ParserFn func(line string) (*values.Result, error)

type Log struct {
	Name           string
	StartsWithTime bool
	Parsers        []ParserFn
}

var ParserFunctions = []Log{
	{
		Name:           "info",
		StartsWithTime: false,
		Parsers: []ParserFn{
			BucketCreated,
			BucketDeleted,
			XDCRReplicationCreatedOrRemovedStart,
		},
	},
	{
		Name:           "diag",
		StartsWithTime: true,
		Parsers: []ParserFn{
			NodeJoinedCluster,
			BucketUpdated,
			BucketFlushed,
			AnalyticsCollectionCreated,
			AnalyticsCollectionDropped,
			AnalyticsIndexCreatedOrDropped,
			AnalyticsScopeCreatedOrDropped,
			LinkConnectedOrDisconnected,
			DataverseCreatedOrDropped,
			DatasetCreated,
			DatasetDropped,
			RebalanceFinish,
			FailoverFinish,
			RebalanceStartTime,
			FailoverStartTime,
			NodeWentDown,
		},
	},
	{
		Name:           "indexer",
		StartsWithTime: true,
		Parsers: []ParserFn{
			IndexCreated,
			IndexDeleted,
			IndexerActive,
		},
	},
	{
		Name:           "goxdcr",
		StartsWithTime: true,
		Parsers: []ParserFn{
			XDCRReplicationCreateOrRemoveFailed,
			XDCRReplicationCreateOrRemoveSuccess,
		},
	},
	{
		Name:           "backup_service",
		StartsWithTime: true,
		Parsers: []ParserFn{
			TaskFinished,
			TaskStarted,
			BackupRemoved,
			BackupPausedOrResumed,
			BackupPlanCreatedOrDeleted,
			BackupRepositoryDeleted,
			BackupRepositoryCreated,
			BackupRepositoryImported,
			BackupRepositoryArchived,
		},
	},
	{
		Name:           "fts",
		StartsWithTime: true,
		Parsers: []ParserFn{
			FTSIndexCreatedOrDropped,
		},
	},
	{
		Name:           "eventing",
		StartsWithTime: true,
		Parsers: []ParserFn{
			EventFunctionDeployedOrUndeployed,
		},
	},
	{
		Name:           "debug",
		StartsWithTime: false,
		Parsers: []ParserFn{
			PasswordPolicyOrLDAPSettingsModified,
			GroupAddedOrRemoved,
			UserAdded,
			UserRemoved,
			AddOrDropScope,
			AddOrDropCollection,
			MinTLSChanged,
		},
	},
}
