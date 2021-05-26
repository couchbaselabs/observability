package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

func TestBucketCreated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `[ns_server:info,2021-02-19T13:19:48.472Z,ns_1@127.0.0.1:ns_memcached-travel-sample<0.6566.0>:ns_memca` +
				`ched:do_ensure_bucket:1240]Created bucket "travel-sample" with config string "max_size=209715200;dbname` +
				`=/opt/couchbase/var/lib/couchbase/data/travel-sample;backend=couchdb;couch_bucket=travel-sample;max_vbuckets` +
				`=1024;alog_path=/opt/couchbase/var/lib/couchbase/data/travel-sample/access.log;data_traffic_enabled=false;` +
				`max_num_workers=3;uuid=b9d054aba12eb8dfcf982fd875b2bb4d;conflict_resolution_type=seqno;bucket_type=persiste` +
				`nt;durability_min_level=none;magma_fragmentation_percentage=50;item_eviction_policy=value_only;persistent_` +
				`metadata_purge_age=259200;max_ttl=0;ht_locks=47;compression_mode=passive;failpartialwarmup=false"`,
			ExpectedResult: &values.Result{
				Event:      values.BucketCreatedEvent,
				Bucket:     "travel-sample",
				BucketType: "persistent",
			},
		},
		{
			Name: "notInLine",
			Line: `[ns_server:info,2021-02-19T13:19:48.472Z,ns_1@127.0.0.1:ns_memcached-travel-sample<0.6566.0>:` +
				`ns_memcached:do_ensure_bucket:1240]Updated bucket "travel-sample" with config string "max_size=209715200;` +
				`dbname=/opt/couchbase/var/lib/couchbase/data/travel-sample;backend=couchdb;couch_bucket=travel-sample;` +
				`max_vbuckets=1024;alog_path=/opt/couchbase/var/lib/couchbase/data/travel-sample`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BucketCreated)
}

func TestBucketDeleted(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `[menelaus:info,2021-03-01T13:17:16.144Z,ns_1@10.144.210.101:<0.27363.671>:menelaus_web_buckets:` +
				`handle_bucket_delete:408]Deleted bucket "beer-sample"`,
			ExpectedResult: &values.Result{
				Event:  values.BucketDeletedEvent,
				Bucket: "beer-sample",
			},
		},
		{
			Name: "notInLine",
			Line: `[menelaus:info,2021-03-01T13:17:16.144Z,ns_1@10.144.210.101:<0.27363.671>:menelaus_web_buckets:` +
				`handle_bucket_delete:408]Deleted "beer-sample"`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BucketDeleted)
}

func TestBucketUpdated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-02-19T13:19:51.351Z, menelaus_web_buckets:0:info:message(ns_1@127.0.0.1) - Updated bucket ` +
				`"travel-sample" (of type couchbase) properties:[{num_replicas,0},{ram_quota,209715200},{flush_enabled,fa` +
				`lse},{storage_mode,couchstore}]`,
			ExpectedResult: &values.Result{
				Event:  values.BucketUpdatedEvent,
				Bucket: "travel-sample",
				Settings: map[string]string{
					"num_replicas":  "0",
					"ram_quota":     "209715200",
					"flush_enabled": "false",
					"storage_mode":  "couchstore",
				},
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T13:19:51.351Z, menelaus_web_buckets:0:info:message(ns_1@127.0.0.1) - Updated ` +
				`"travel-sample" (of type couchbase) properties:
			[{num_replicas,0},
			 {ram_quota,209715200},
			 {flush_enabled,false},
			 {storage_mode,couchstore}]`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BucketUpdated)
}

func TestBucketFlushed(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-03-15T09:55:06.565Z, ns_orchestrator:0:info:message(ns_1@10.144.210.101) - Flushing bucket "test" ` +
				`from node 'ns_1@10.144.210.101'`,
			ExpectedResult: &values.Result{
				Event:  values.BucketFlushedEvent,
				Bucket: "test",
				Node:   "ns_1@10.144.210.101",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-15T09:55:06.565Z, ns_orchestrator:0:info:message(ns_1@10.144.210.101) - Create bucket "test"` +
				` from node 'ns_1@10.144.210.101'`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, BucketFlushed)
}
