package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

func TestXDCRReplicationCreatedOrRemovedStart(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineXDCRReplicationCreationStart",
			Line: `[user:info,2021-03-03T15:56:40.386Z,ns_1@10.144.210.101:<0.32519.916>:xdcr:unknown:-1]Replication ` +
				`from bucket "test" to bucket "target" on cluster "remote" created.`,
			ExpectedResult: &values.Result{
				Event:        values.XDCRReplicationCreateStartedEvent,
				SourceBucket: "test",
				TargetBucket: "target",
				Cluster:      "remote",
			},
		},
		{
			Name: "inLineXDCRReplicationRemoved",
			Line: `[user:info,2021-03-03T15:58:45.247Z,ns_1@10.144.210.101:<0.15154.917>:xdcr:unknown:-1]Replication from` +
				` bucket "test" to bucket "target" on cluster "remote" removed.`,
			ExpectedResult: &values.Result{
				Event:        values.XDCRReplicationRemoveStartedEvent,
				SourceBucket: "test",
				TargetBucket: "target",
				Cluster:      "remote",
			},
		},
		{
			Name: "notInLine",
			Line: `[info,2021-03-03T15:58:45.247Z,ns_1@10.144.210.101:<0.15154.917>:xdcr:unknown:-1]Replication` +
				` to bucket "test" to bucket "target" on cluster "remote" alterd.`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, XDCRReplicationCreatedOrRemovedStart)
}

func TestXDCRReplicationCreateOrRemoveFailed(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-03-03T15:56:11.715Z ERRO GOXDCR.AdminPort: Error creating replication. errorsMap=map[toBucket:` +
				`Error validating target bucket 'target'. err=BucketValidationInfo Operation failed after max retries.  ` +
				`Last error: Bucket doesn't exist]`,
			ExpectedResult: &values.Result{
				Event:  values.XDCRReplicationCreateFailedEvent,
				Reason: "Bucket doesn't exist",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-03T15:56:11.715Z ERRO GOXDCR.AdminPort: Error in replication. errorsMap=map[toBucket:Error` +
				`validating target bucket 'target'. err=BucketValidationInfo Operation failed after max retries.  Last error: ` +
				`Bucket doesn't exist]`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, XDCRReplicationCreateOrRemoveFailed)
}

func TestXDCRReplicationCreateOrRemoveSuccess(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineCreate",
			Line: `2021-03-03T15:56:40.372Z INFO GOXDCR.ReplMgr: Replication specification 20768cf022f910817e9dbdf` +
				`19b1fe837/test/target is created`,
			ExpectedResult: &values.Result{
				Event: values.XDCRReplicationCreateSuccessfulEvent,
			},
		},
		{
			Name: "inLineRemove",
			Line: `2021-03-03T15:58:45.244Z INFO GOXDCR.ReplMgr: Replication specification 20768cf022f910817e9dbdf19b` +
				`1fe837/test/target is deleted`,
			ExpectedResult: &values.Result{
				Event: values.XDCRReplicationRemoveSuccessfulEvent,
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-03T15:56:40.372Z INFO GOXDCR.ReplMgr: Replication specification 20768cf022f910817e9dbdf19` +
				`b1fe837/test/target create`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, XDCRReplicationCreateOrRemoveSuccess)
}
