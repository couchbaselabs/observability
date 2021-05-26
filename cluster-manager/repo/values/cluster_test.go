package values

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCouchbaseClusterGetTLSConfig(t *testing.T) {
	t.Run("no-ca", func(t *testing.T) {
		cluster := &CouchbaseCluster{
			UUID:     "uuid-1",
			User:     "user",
			Password: "password",
		}

		tlsConfig := cluster.GetTLSConfig()
		if tlsConfig == nil {
			t.Fatal("Expected a config got <nil>")
		}

		if !tlsConfig.InsecureSkipVerify {
			t.Fatalf("Expected skip verify to be true got false")
		}
	})

	t.Run("ca", func(t *testing.T) {
		cluster := &CouchbaseCluster{
			UUID:     "uuid-1",
			User:     "user",
			Password: "password",
		}

		caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			t.Fatalf("could not create key for ca: %v", err)
		}

		ca := &x509.Certificate{
			SerialNumber: big.NewInt(2019),
			Subject: pkix.Name{
				Organization:  []string{"Company, INC."},
				Country:       []string{"US"},
				Province:      []string{""},
				Locality:      []string{"San Francisco"},
				StreetAddress: []string{"Golden Gate Bridge"},
				PostalCode:    []string{"94016"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			IsCA:                  true,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
		}

		caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
		if err != nil {
			t.Fatalf("could not create certificate: %v", err)
		}

		cluster.CaCert = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caBytes,
		})

		tlsConfig := cluster.GetTLSConfig()
		if tlsConfig == nil {
			t.Fatal("Expected a config got <nil>")
		}

		if tlsConfig.InsecureSkipVerify {
			t.Fatalf("Expected skip verify to be false got true")
		}
	})
}

func TestNodesSummaryGetHosts(t *testing.T) {
	type testCase struct {
		name string
		in   NodesSummary
		out  []string
	}

	cases := []testCase{
		{
			name: "nil",
			out:  []string{},
		},
		{
			name: "empty",
			in:   []NodeSummary{},
			out:  []string{},
		},
		{
			name: "values",
			in:   NodesSummary{{Host: "http://localhost:9000"}, {Host: "http://localhost:9001"}},
			out:  []string{"http://localhost:9000", "http://localhost:9001"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out := tc.in.GetHosts(); !reflect.DeepEqual(out, tc.out) {
				t.Fatalf("Expected %#v got %#v", tc.out, out)
			}
		})
	}
}

func TestBucketsSummaryGetBucketNames(t *testing.T) {
	type testCase struct {
		name string
		in   BucketsSummary
		out  []string
	}

	cases := []testCase{
		{
			name: "nil",
			out:  []string{},
		},
		{
			name: "empty",
			in:   BucketsSummary{},
			out:  []string{},
		},
		{
			name: "values",
			in:   BucketsSummary{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			out:  []string{"a", "b", "c"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out := tc.in.GetBucketNames(); !reflect.DeepEqual(out, tc.out) {
				t.Fatalf("Expected %#v got %#v", tc.out, out)
			}
		})
	}
}

func TestMarshallBucketsSummaryFromRest(t *testing.T) {
	reader := bytes.NewReader([]byte(`[{"name":"empty","nodeLocator":"vbucket",
"uuid":"1f872e36970f76c3392c8d8edf4299d9",
"uri":"/pools/default/buckets/empty?bucket_uuid=1f872e36970f76c3392c8d8edf4299d9",
"streamingUri":"/pools/default/bucketsStreaming/empty?bucket_uuid=1f872e36970f76c3392c8d8edf4299d9",
"bucketCapabilitiesVer":"","bucketCapabilities":["collections","durableWrite","tombstonedUserXAttrs","couchapi",
"subdoc.ReplaceBodyWithXattr","subdoc.DocumentMacroSupport","dcp","cbhello","touch","cccp","xdcrCheckpointing",
"nodesExt","xattr"],"collectionsManifestUid":"0","ddocs":{"uri":"/pools/default/buckets/empty/ddocs"},
"bucketType":"membase","authType":"sasl","localRandomKeyUri":"/pools/default/buckets/empty/localRandomKey",
"controllers":{"compactAll":"/pools/default/buckets/empty/controller/compactBucket",
"compactDB":"/pools/default/buckets/empty/controller/compactDatabases",
"purgeDeletes":"/pools/default/buckets/empty/controller/unsafePurgeBucket",
"startRecovery":"/pools/default/buckets/empty/controller/startRecovery"},
"nodes":[{"couchApiBaseHTTPS":"https://127.0.0.1:19500/empty%2B1f872e36970f76c3392c8d8edf4299d9",
"couchApiBase":"http://127.0.0.1:9500/empty%2B1f872e36970f76c3392c8d8edf4299d9","clusterMembership":"active",
"recoveryType":"none","status":"healthy","otpNode":"n_0@127.0.0.1","thisNode":true,"hostname":"127.0.0.1:9000",
"nodeUUID":"fb8c662e7c894424f0c6a61cad8474a8","clusterCompatibility":458752,"version":"0.0.0-0000-enterprise",
"os":"x86_64-apple-darwin18.7.0","cpuCount":12,"ports":{"direct":12000,"httpsCAPI":19500,"httpsMgmt":19000,
"distTCP":21400,"distTLS":21450},"services":["kv"],"nodeEncryption":false,"addressFamilyOnly":false,
"configuredHostname":"127.0.0.1:9000","addressFamily":"inet","externalListeners":[{"afamily":"inet",
"nodeEncryption":false}],"replication":0,"systemStats":{"cpu_utilization_rate":16.49175412293853,
"cpu_stolen_rate":0,"swap_total":7516192768,"swap_used":6375342080,"mem_total":34359738368,"mem_free":13161459712,
"mem_limit":34359738368,"cpu_cores_available":12,"allocstall":18446744073709552000},
"interestingStats":{"couch_docs_actual_disk_size":61777557,
"couch_views_actual_disk_size":0,"curr_items":63182,"curr_items_tot":63182,"ep_bg_fetched":0,
"couch_docs_data_size":46104353,"mem_used":139047272,"vb_active_num_non_resident":0,"vb_replica_curr_items":0,
"cmd_get":0,"get_hits":0,"ops":0,"couch_spatial_disk_size":0,"couch_views_data_size":0,"couch_spatial_data_size":0},
"uptime":"5792","memoryTotal":34359738368,"memoryFree":13161459712,"mcdMemoryReserved":26214,
"mcdMemoryAllocated":26214}],"stats":{"uri":"/pools/default/buckets/empty/stats",
"directoryURI":"/pools/default/buckets/empty/stats/Directory","nodeStatsListURI":"/pools/default/buckets/empty/nodes"},
"autoCompactionSettings":false,"replicaIndex":false,"replicaNumber":1,"threadsNumber":3,"quota":{"ram":314572800,
"rawRAM":314572800},"basicStats":{"quotaPercentUsed":11.18166097005208,"opsPerSec":0,"diskFetches":0,"itemCount":0,
"diskUsed":1724245,"dataUsed":100332,"memUsed":35174464,"vbActiveNumNonResident":0},"evictionPolicy":"valueOnly",
"storageBackend":"couchstore","durabilityMinLevel":"none","pitrEnabled":false,"pitrGranularity":600,
"pitrMaxHistoryAge":86400,"fragmentationPercentage":50,"conflictResolutionType":"seqno","maxTTL":0,
"compressionMode":"active","saslPassword":"734ca761167d08e56780f8d058ed74e8"},{"name":"travel-sample",
"nodeLocator":"vbucket","uuid":"6f89e48359a839bb1f31d87228343a63",
"uri":"/pools/default/buckets/travel-sample?bucket_uuid=6f89e48359a839bb1f31d87228343a63",
"streamingUri":"/pools/default/bucketsStreaming/travel-sample?bucket_uuid=6f89e48359a839bb1f31d87228343a63",
"bucketCapabilitiesVer":"","bucketCapabilities":["collections","durableWrite","tombstonedUserXAttrs","couchapi",
"subdoc.ReplaceBodyWithXattr","subdoc.DocumentMacroSupport","dcp","cbhello","touch","cccp","xdcrCheckpointing",
"nodesExt","xattr"],"collectionsManifestUid":"1","ddocs":{"uri":"/pools/default/buckets/travel-sample/ddocs"},
"bucketType":"membase","authType":"sasl","localRandomKeyUri":"/pools/default/buckets/travel-sample/localRandomKey",
"controllers":{"compactAll":"/pools/default/buckets/travel-sample/controller/compactBucket",
"compactDB":"/pools/default/buckets/travel-sample/controller/compactDatabases",
"flush": "/pools/default/buckets/travel-sample/controller/doFlush",
"purgeDeletes":"/pools/default/buckets/travel-sample/controller/unsafePurgeBucket",
"startRecovery":"/pools/default/buckets/travel-sample/controller/startRecovery"},
"nodes":[{"couchApiBaseHTTPS":"https://127.0.0.1:19500/travel-sample%2B6f89e48359a839bb1f31d87228343a63",
"couchApiBase":"http://127.0.0.1:9500/travel-sample%2B6f89e48359a839bb1f31d87228343a63","clusterMembership":"active",
"recoveryType":"none","status":"healthy","otpNode":"n_0@127.0.0.1","thisNode":true,"hostname":"127.0.0.1:9000",
"nodeUUID":"fb8c662e7c894424f0c6a61cad8474a8","clusterCompatibility":458752,"version":"0.0.0-0000-enterprise",
"os":"x86_64-apple-darwin18.7.0","cpuCount":12,"ports":{"direct":12000,"httpsCAPI":19500,"httpsMgmt":19000,
"distTCP":21400,"distTLS":21450},"services":["kv"],"nodeEncryption":false,"addressFamilyOnly":false,
"configuredHostname":"127.0.0.1:9000","addressFamily":"inet","externalListeners":[{"afamily":"inet",
"nodeEncryption":false}],"replication":0,"systemStats":{"cpu_utilization_rate":16.49175412293853,"cpu_stolen_rate":0,
"swap_total":7516192768,"swap_used":6375342080,"mem_total":34359738368,"mem_free":13161459712,"mem_limit":34359738368,
"cpu_cores_available":12,"allocstall":18446744073709552000},"interestingStats":{"couch_docs_actual_disk_size":61777557,
"couch_views_actual_disk_size":0,"curr_items":63182,"curr_items_tot":63182,"ep_bg_fetched":0,
"couch_docs_data_size":46104353,"mem_used":139047272,"vb_active_num_non_resident":0,"vb_replica_curr_items":0,
"cmd_get":0,"get_hits":0,"ops":0,"couch_spatial_disk_size":0,"couch_views_data_size":0,"couch_spatial_data_size":0},
"uptime":"5792","memoryTotal":34359738368,"memoryFree":13161459712,"mcdMemoryReserved":26214,
"mcdMemoryAllocated":26214}],"stats":{"uri":"/pools/default/buckets/travel-sample/stats",
"directoryURI":"/pools/default/buckets/travel-sample/stats/Directory",
"nodeStatsListURI":"/pools/default/buckets/travel-sample/nodes"},"autoCompactionSettings":false,
"replicaIndex":false,"replicaNumber":1,"threadsNumber":3,"quota":{"ram":209715200,"rawRAM":209715200},
"basicStats":{"quotaPercentUsed":49.53041458129883,"opsPerSec":0,"diskFetches":0,"itemCount":63182,"diskUsed":60053312,
"dataUsed":46004021,"memUsed":103872808,"vbActiveNumNonResident":0},"evictionPolicy":"fullEviction",
"storageBackend":"couchstore","durabilityMinLevel":"none","pitrEnabled":false,"pitrGranularity":600,
"pitrMaxHistoryAge":86400,"fragmentationPercentage":50,"conflictResolutionType":"seqno","maxTTL":0,
"compressionMode":"passive","saslPassword":"0c1e07679db78d2746b954347848470e"}]`))

	summary, err := MarshallBucketsSummaryFromRest(reader)
	require.NoError(t, err)

	expected := BucketsSummary{
		{
			Name:                   "empty",
			CompressionMode:        "active",
			ConflictResolutionType: "seqno",
			BucketType:             "couchbase",
			StorageBackend:         "couchstore",
			EvictionPolicy:         "valueOnly",
			Quota:                  314572800,
			QuotaUsed:              11.18166097005208,
			FlushEnabled:           false,
			NumReplicas:            1,
			Items:                  0,
		},
		{
			Name:                   "travel-sample",
			CompressionMode:        "passive",
			ConflictResolutionType: "seqno",
			BucketType:             "couchbase",
			StorageBackend:         "couchstore",
			EvictionPolicy:         "fullEviction",
			Quota:                  209715200,
			QuotaUsed:              49.53041458129883,
			FlushEnabled:           true,
			NumReplicas:            1,
			Items:                  63182,
		},
	}

	require.Equal(t, expected, summary)
}

func TestParseVersions(t *testing.T) {
	reader := []byte(`{"versions": [{"Version": "5.5.0-2958","EOM": "2020-07-01",
"EOS": "2021-10-01","OS": ["Microsoft Windows Server 2012 R2 Standard",
"Microsoft Windows Server 2016 Standard","Amazon Linux AMI release 2017.09","Amazon Linux AMI release 2018.03",
"CentOS release 6.","CentOS Linux release 7.","Debian GNU/Linux 8.","Debian GNU/Linux 9.","openSUSE 11.",
"openSUSE 12.","SUSE Linux Enterprise Server 11","SUSE Linux Enterprise Server 12",
"Oracle Linux Server release 6.","Oracle Linux Server release 7.","Red Hat Enterprise Linux Server release 6.",
"Red Hat Enterprise Linux Server release 7.","Ubuntu 14.","Ubuntu 16."]}]}`)

	summary, err := parseVersions(reader)
	require.NoError(t, err)

	eomTime, err := time.Parse("2006-01-02", "2020-07-01")
	require.NoError(t, err)
	eosTime, err := time.Parse("2006-01-02", "2021-10-01")
	require.NoError(t, err)

	expected := Versions{
		"5.5.0-2958": {
			Build: "5.5.0-2958",
			EOM:   eomTime,
			EOS:   eosTime,
			OS: []string{
				"Microsoft Windows Server 2012 R2 Standard", "Microsoft Windows Server 2016 Standard",
				"Amazon Linux AMI release 2017.09", "Amazon Linux AMI release 2018.03", "CentOS release 6.",
				"CentOS Linux release 7.", "Debian GNU/Linux 8.", "Debian GNU/Linux 9.", "openSUSE 11.",
				"openSUSE 12.", "SUSE Linux Enterprise Server 11",
				"SUSE Linux Enterprise Server 12", "Oracle Linux Server release 6.",
				"Oracle Linux Server release 7.", "Red Hat Enterprise Linux Server release 6.",
				"Red Hat Enterprise Linux Server release 7.", "Ubuntu 14.", "Ubuntu 16.",
			},
		},
	}

	require.Equal(t, expected, summary)
}
