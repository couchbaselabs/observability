package values

type EventType string

const (
	DatasetCreatedEvent        EventType = "dataset_created"
	DatasetDroppedEvent        EventType = "dataset_dropped"
	AnalyticsIndexCreatedEvent EventType = "analytics_index_created"
	AnalyticsIndexDroppedEvent EventType = "analytics_index_dropped"

	TaskFinishedEvent             EventType = "task_finished"
	TaskStartedEvent              EventType = "task_started"
	BackupRemovedEvent            EventType = "backup_removed"
	BackupPausedEvent             EventType = "backup_paused"
	BackupResumedEvent            EventType = "backup_resumed"
	BackupPlanCreatedEvent        EventType = "backup_plan_created"
	BackupPlanDeletedEvent        EventType = "backup_plan_deleted"
	BackupRepositoryDeletedEvent  EventType = "backup_repo_created"
	BackupRepositoryCreatedEvent  EventType = "backup_repo_deleted"
	BackupRepositoryImportedEvent EventType = "backup_repo_imported"
	BackupRepositoryArchivedEvent EventType = "backup_repo_archived"

	RebalanceStartEvent  EventType = "rebalance_start"
	RebalanceFinishEvent EventType = "rebalance_finish"
	FailoverStartEvent   EventType = "failover_start"
	FailoverEndEvent     EventType = "failover_end"
	NodeJoinedEvent      EventType = "node_joined"
	NodeWentDownEvent    EventType = "node_went_down"

	EventingFunctionDeployedEvent   EventType = "eventing_function_deployed"
	EventingFunctionUndeployedEvent EventType = "eventing_function_undeployed"

	FTSIndexCreatedEvent EventType = "fts_index_created"
	FTSIndexDroppedEvent EventType = "fts_index_dropped"

	IndexCreatedEvent  EventType = "index_created"
	IndexDeletedEvent  EventType = "index_deleted"
	IndexerActiveEvent EventType = "indexer_active"
	IndexBuiltEvent    EventType = "index_built"

	BucketCreatedEvent EventType = "bucket_created"
	BucketDeletedEvent EventType = "bucket_deleted"
	BucketUpdatedEvent EventType = "bucket_updated"
	BucketFlushedEvent EventType = "bucket_flushed"

	DroppedTicksEvent           EventType = "dropped_ticks"
	DataLostEvent               EventType = "data_lost"
	ServerErrorEvent            EventType = "server_error"
	SigkillErrorEvent           EventType = "sigkill_error"
	LostConnectionToServerEvent EventType = "lost_connection_to_server"

	LDAPSettingsModifiedEvent  EventType = "LDAP_settings_modified"
	PasswordPolicyChangedEvent EventType = "password_policy_changed"
	GroupAddedEvent            EventType = "group_added"
	GroupDeletedEvent          EventType = "group_deleted"
	UserAddedEvent             EventType = "user_added"
	UserDeletedEvent           EventType = "user_deleted"

	XDCRReplicationCreateStartedEvent    EventType = "XDCR_replication_create_started"
	XDCRReplicationRemoveStartedEvent    EventType = "XDCR_replication_remove_started"
	XDCRReplicationCreateFailedEvent     EventType = "XDCR_replication_create_failed"
	XDCRReplicationCreateSuccessfulEvent EventType = "XDCR_replication_create_successful"
	XDCRReplicationCreatedEvent          EventType = "XDCR_replication_created"
	XDCRReplicationRemoveFailedEvent     EventType = "XDCR_replication_remove_failed"
	XDCRReplicationRemoveSuccessfulEvent EventType = "XDCR_replication_remove_successful"
)

var EventTypes = map[string]EventType{
	"dataset_created":         DatasetCreatedEvent,
	"dataset_dropped":         DatasetDroppedEvent,
	"analytics_index_created": AnalyticsIndexCreatedEvent,
	"analytics_index_dropped": AnalyticsIndexDroppedEvent,

	"task_finished":        TaskFinishedEvent,
	"task_started":         TaskStartedEvent,
	"backup_removed":       BackupRemovedEvent,
	"backup_paused":        BackupPausedEvent,
	"backup_resumed":       BackupResumedEvent,
	"backup_plan_created":  BackupPlanCreatedEvent,
	"backup_plan_deleted":  BackupPlanDeletedEvent,
	"backup_repo_created":  BackupRepositoryDeletedEvent,
	"backup_repo_deleted":  BackupRepositoryCreatedEvent,
	"backup_repo_imported": BackupRepositoryImportedEvent,
	"backup_repo_archived": BackupRepositoryArchivedEvent,

	"rebalance_start":  RebalanceStartEvent,
	"rebalance_finish": RebalanceFinishEvent,
	"failover_start":   FailoverStartEvent,
	"failover_end":     FailoverEndEvent,
	"node_joined":      NodeJoinedEvent,
	"node_went_down":   NodeWentDownEvent,

	"eventing_function_deployed":   EventingFunctionDeployedEvent,
	"eventing_function_undeployed": EventingFunctionUndeployedEvent,

	"fts_index_created": FTSIndexCreatedEvent,
	"fts_index_dropped": FTSIndexDroppedEvent,

	"index_created":  IndexCreatedEvent,
	"index_deleted":  IndexDeletedEvent,
	"indexer_active": IndexerActiveEvent,
	"index_built":    IndexBuiltEvent,

	"bucket_created": BucketCreatedEvent,
	"bucket_deleted": BucketDeletedEvent,
	"bucket_updated": BucketUpdatedEvent,
	"bucket_flushed": BucketFlushedEvent,

	"dropped_ticks":             DroppedTicksEvent,
	"data_lost":                 DataLostEvent,
	"server_error":              ServerErrorEvent,
	"sigkill_error":             SigkillErrorEvent,
	"lost_connection_to_server": LostConnectionToServerEvent,

	"LDAP_settings_modified":  LDAPSettingsModifiedEvent,
	"password_policy_changed": PasswordPolicyChangedEvent,
	"group_added":             GroupAddedEvent,
	"group_deleted":           GroupDeletedEvent,
	"user_added":              UserAddedEvent,
	"user_deleted":            UserDeletedEvent,

	"XDCR_replication_create_started":    XDCRReplicationCreateStartedEvent,
	"XDCR_replication_remove_started":    XDCRReplicationRemoveStartedEvent,
	"XDCR_replication_create_failed":     XDCRReplicationCreateFailedEvent,
	"XDCR_replication_create_successful": XDCRReplicationCreateSuccessfulEvent,
	"XDCR_replication_created":           XDCRReplicationCreatedEvent,
	"XDCR_replication_remove_failed":     XDCRReplicationRemoveFailedEvent,
	"XDCR_replication_remove_successful": XDCRReplicationRemoveSuccessfulEvent,
}
