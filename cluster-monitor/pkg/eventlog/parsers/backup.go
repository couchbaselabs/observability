// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"encoding/json"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

// TaskFinished gets the status of a finished backup, merge or restore task.
func TaskFinished(line string) (*values.Result, error) {
	if !strings.Contains(line, "Task finished") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repository"`
		Status     string `json:"status"`
		Task       string `json:"task"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event:      values.TaskFinishedEvent,
		Successful: dest.Status == "done",
		Task:       dest.Task,
		Repo:       dest.Repository,
		Reason:     dest.Status,
	}, nil
}

// TaskStarted gets when a backup, merge or restore task starts.
func TaskStarted(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Worker) Running task") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repositoryID"`
		Task       string `json:"taskName"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event: values.TaskStartedEvent,
		Task:  dest.Task,
		Repo:  dest.Repository,
	}, nil
}

// BackupRemoved gets when a backup is deleted.
func BackupRemoved(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Manager) Backup deleted") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repo"`
		Backup     string `json:"backup"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event:  values.BackupRemovedEvent,
		Repo:   dest.Repository,
		Backup: dest.Backup,
	}, nil
}

// BackupPausedOrResumed gets when a backup repository is paused or resumed.
func BackupPausedOrResumed(line string) (*values.Result, error) {
	if !strings.Contains(line, "Paused repository tasks") && !strings.Contains(line, "Resumed repository tasks") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repository"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	event := values.BackupResumedEvent
	if strings.Contains(line, "Paused repository tasks") {
		event = values.BackupPausedEvent
	}

	return &values.Result{
		Event: event,
		Repo:  dest.Repository,
	}, nil
}

// BackupPlanCreatedOrDeleted gets when a backup plan is created or deleted.
func BackupPlanCreatedOrDeleted(line string) (*values.Result, error) {
	if !strings.Contains(line, "(HTTP Manager) Added plan") && !strings.Contains(line, "(Manager) Plan deleted") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Plan string `json:"name"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	event := values.BackupPlanCreatedEvent
	if strings.Contains(line, "(Manager) Plan deleted") {
		event = values.BackupPlanDeletedEvent
	}

	return &values.Result{
		Event: event,
		Plan:  dest.Plan,
	}, nil
}

// BackupRepositoryDeleted gets when a backup repository is deleted.
func BackupRepositoryDeleted(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Manager) Repository deleted") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repo string `json:"id"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event: values.BackupRepositoryDeletedEvent,
		Repo:  dest.Repo,
	}, nil
}

// BackupRepositoryCreated gets when a backup repository is created.
func BackupRepositoryCreated(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Manger) Added repository") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repo"`
		Plan       string `json:"plan"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event: values.BackupRepositoryCreatedEvent,
		Repo:  dest.Repository,
		Plan:  dest.Plan,
	}, nil
}

// BackupRepositoryImported gets when a backup repository is imported.
func BackupRepositoryImported(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Manger) Imported repository") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		Repository string `json:"repo"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event: values.BackupRepositoryImportedEvent,
		Repo:  dest.Repository,
	}, nil
}

// BackupRepositoryArchived gets when a backup repository is archived.
func BackupRepositoryArchived(line string) (*values.Result, error) {
	if !strings.Contains(line, "(Manger) Repository archived") {
		return nil, values.ErrNotInLine
	}

	type repoInfo struct {
		OldRepo string `json:"oldID"`
		NewRepo string `json:"newID"`
	}

	var dest repoInfo
	if err := unmarshalBackupServiceJSON(line, &dest); err != nil {
		return nil, err
	}

	return &values.Result{
		Event:         values.BackupRepositoryArchivedEvent,
		OldRepository: dest.OldRepo,
		NewRepository: dest.NewRepo,
	}, nil
}

// unmarshalBackupServiceJSON unmarshals backup information from a json string to a given interface.
func unmarshalBackupServiceJSON(line string, dest interface{}) error {
	jsonDoc := line[strings.Index(line, "{"):]
	return json.Unmarshal([]byte(jsonDoc), &dest)
}
