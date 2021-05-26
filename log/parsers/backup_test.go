package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"

	"github.com/stretchr/testify/require"
)

type inLineTestCase struct {
	Name           string
	Line           string
	ExpectedResult *values.Result
	ExpectedError  error
}

func TestTaskFinished(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupFinished",
			Line: `2021-03-04T15:51:17.568Z	INFO	(Event Handler) (Node Run) Task finished  {"cluster": "self", ` +
				`"repository": "simple", "task": "BACKUP-142a8e59-c3d3-4082-8195-6529e9b15a16", "status": "done"}`,
			ExpectedResult: &values.Result{
				Event:      values.TaskFinishedEvent,
				Successful: true,
				Task:       "BACKUP-142a8e59-c3d3-4082-8195-6529e9b15a16",
				Repo:       "simple",
				Reason:     "done",
			},
			ExpectedError: nil,
		},
		{
			Name: "mergeFinished",
			Line: `2021-03-04T15:52:33.592Z	INFO	(Event Handler) (Node Run) Task finished	{"cluster": "self", ` +
				`"repository": "simple", "task": "MERGE-3dc3ee49-fb32-4a76-98aa-51f10c597294", "status": "done"}`,
			ExpectedResult: &values.Result{
				Event:      values.TaskFinishedEvent,
				Successful: true,
				Task:       "MERGE-3dc3ee49-fb32-4a76-98aa-51f10c597294",
				Repo:       "simple",
				Reason:     "done",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-04T15:51:17.568Z	INFO	(Event Handler) (Node Run) Task	{"cluster": "self",` +
				`"repository": "simple", "task": "MERGE-142a8e59-c3d3-4082-8195-6529e9b15a16", "status": "done"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, TaskFinished)
}

func TestTaskStarted(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupStart",
			Line: `2021-03-04T15:51:15.908Z	INFO	(Worker) Running task	{"cluster": "self", "repositoryID": "simple", ` +
				`"state": "active", "taskName": "BACKUP-142a8e59-c3d3-4082-8195-6529e9b15a16"}`,
			ExpectedResult: &values.Result{
				Event: values.TaskStartedEvent,
				Task:  "BACKUP-142a8e59-c3d3-4082-8195-6529e9b15a16",
				Repo:  "simple",
			},
		},
		{
			Name: "mergeStart",
			Line: `2021-03-04T15:52:15.203Z	INFO	(Worker) Running task	{"cluster": "self", "repositoryID": "simple", ` +
				`"state": "active", "taskName": "MERGE-3dc3ee49-fb32-4a76-98aa-51f10c597294"}`,
			ExpectedResult: &values.Result{
				Event: values.TaskStartedEvent,
				Task:  "MERGE-3dc3ee49-fb32-4a76-98aa-51f10c597294",
				Repo:  "simple",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-10T08:59:59.225Z	INFO	(Dispatcher) Set task	{"task": {"task_name":"merge_week",` +
				`"status":"unknown","start":"2021-03-10T08:59:59.098932829Z","end":"0001-01-01T00:00:00Z","error_code":0` +
				`,"type":"MERGE"}}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, TaskStarted)
}

func TestBackupRemoved(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupRemoved",
			Line: `2021-03-29T11:58:18.734Z        INFO    (Manager) Backup deleted        {"cluster": "self", "repo": ` +
				`"simple", "backup": "2021-03-29T11_23_29.963441Z"}`,
			ExpectedResult: &values.Result{
				Event:  values.BackupRemovedEvent,
				Repo:   "simple",
				Backup: "2021-03-29T11_23_29.963441Z",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-11T12:44:25.559Z	DEBUG	(REST) DELETE /api/v1/cluster/self/repository/active/simple/backups/` +
				`2021-03-11T09_03_41.864801195Z Authenticated user: Administrator`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupRemoved)
}

func TestBackupPausedOrResumed(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineBackupPaused",
			Line: `2021-03-17T16:40:39.846Z	INFO	(Clockkeper) Paused repository tasks	{"cluster": "self", "repository":` +
				`"simple", "#tasks": 0}`,
			ExpectedResult: &values.Result{
				Event: values.BackupPausedEvent,
				Repo:  "simple",
			},
		},
		{
			Name: "inLineBackupResumed",
			Line: `2021-03-17T16:40:44.612Z	INFO	(Clockkeper) Resumed repository tasks	` +
				`{"cluster": "self", "repository": "simple", "#tasks": 9}`,
			ExpectedResult: &values.Result{
				Event: values.BackupResumedEvent,
				Repo:  "simple",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-17T16:40:39.846Z	INFO	(Clockkeper) Merged repository tasks	{"cluster": "self", ` +
				`"repository": "simple", "#tasks": 0}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupPausedOrResumed)
}

func TestBackupPlanCreatedOrDeleted(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupPlanCreated",
			Line: `2021-05-17T10:47:30.187+01:00 INFO (HTTP Manager) Added plan {"name": "ghmcgmcmh"}`,
			ExpectedResult: &values.Result{
				Event: values.BackupPlanCreatedEvent,
				Plan:  "ghmcgmcmh",
			},
		},
		{
			Name: "backupPlanDeleted",
			Line: `2021-05-17T12:59:44.078+01:00 INFO (Manager) Plan deleted {"name": "ghmcgmcmh"}`,
			ExpectedResult: &values.Result{
				Event: values.BackupPlanDeletedEvent,
				Plan:  "ghmcgmcmh",
			},
		},
		{
			Name:          "notInLine",
			Line:          `2021-05-17T10:47:30.187+01:00 INFO (HTTP Manager) Plan added {"name": "ghmcgmcmh"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupPlanCreatedOrDeleted)
}

func TestBackupRepositoryDeleted(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupRepositoryDeleted",
			Line: `2021-05-17T10:48:49.136+01:00 INFO (Manager) Repository deleted {"id": "simple", "state": "archived", ` +
				`"cluster": "self"}`,
			ExpectedResult: &values.Result{
				Event: values.BackupRepositoryDeletedEvent,
				Repo:  "simple",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-05-17T10:48:49.136+01:00 INFO (Manager) Repo deleted {"id": "simple", "state": "archived", ` +
				`"cluster": "self"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupRepositoryDeleted)
}

func TestBackupRepositoryCreated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupRepositoryCreated",
			Line: `2021-05-17T10:47:03.399+01:00 INFO (Manger) Added repository {"id": {"id":"simple","plan_name":` +
				`"_daily_backups","state":"active","archive":"/Users/rebeccacritchlow/data/backup","repo":` +
				`"0b891a2f-5daf-4615-b096-27c8f85e598f","version":1,"creation_time":"0001-01-01T00:00:00Z","update_time":` +
				`"0001-01-01T00:00:00Z"}, "cluster": "self", "archive": "/Users/rebeccacritchlow/data/backup", ` +
				`"repo": "0b891a2f-5daf-4615-b096-27c8f85e598f", "plan": "_daily_backups"}`,
			ExpectedResult: &values.Result{
				Event: values.BackupRepositoryCreatedEvent,
				Repo:  "0b891a2f-5daf-4615-b096-27c8f85e598f",
				Plan:  "_daily_backups",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-05-17T10:47:03.399+01:00 INFO (Manger) Added repo {"id": {"id":"simple","plan_name":` +
				`"_daily_backups","state":"active","archive":"/Users/rebeccacritchlow/data/backup","repo":` +
				`"0b891a2f-5daf-4615-b096-27c8f85e598f","version":1,"creation_time":"0001-01-01T00:00:00Z","update_time":` +
				`"0001-01-01T00:00:00Z"}, "cluster": "self", "archive": "/Users/rebeccacritchlow/data/backup", ` +
				`"repo": "0b891a2f-5daf-4615-b096-27c8f85e598f", "plan": "_daily_backups"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupRepositoryCreated)
}

func TestBackupRepositoryImported(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupRepositoryImported",
			Line: `2021-05-17T10:50:21.726+01:00 INFO (Manger) Imported repository {"cluster": "self", "id": "simple", ` +
				`"archive": "/Users/rebeccacritchlow/data/backups", "repo": "example"}`,
			ExpectedResult: &values.Result{
				Event: values.BackupRepositoryImportedEvent,
				Repo:  "example",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-05-17T10:50:21.726+01:00 INFO (Manger) Imported repo {"cluster": "self", "id": "simple", ` +
				`"archive": "/Users/rebeccacritchlow/data/backups", "repo": "example"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupRepositoryImported)
}

func TestBackupRepositoryArchived(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "backupRepositoryArchived",
			Line: `2021-05-17T10:48:34.104+01:00 INFO (Manger) Repository archived {"cluster": "self", "oldID": "simple", ` +
				`"newID": "new-simple"}`,
			ExpectedResult: &values.Result{
				Event:         values.BackupRepositoryArchivedEvent,
				OldRepository: "simple",
				NewRepository: "new-simple",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-05-17T10:48:34.104+01:00 INFO (Manger) Repo archived {"cluster": "self", "oldID": "simple", ` +
				`"newID": "new-simple"}`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BackupRepositoryArchived)
}

func runTestCases(t *testing.T, testCases []inLineTestCase, fn ParserFn) {
	for _, x := range testCases {
		t.Run(x.Name, func(t *testing.T) {
			inLineResult, err := fn(x.Line)
			require.Equal(t, x.ExpectedResult, inLineResult)
			require.ErrorIs(t, err, x.ExpectedError)
		})
	}
}
