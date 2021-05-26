package parsers

import (
	"github.com/couchbaselabs/cbmultimanager/log/values"
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
			RebalanceFinish,
			FailoverFinish,
			RebalanceStartTime,
			FailoverStartTime,
			BucketCreated,
			BucketDeleted,
			NodeWentDown,
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
		},
	},
}
