package manager

import "github.com/couchbaselabs/observability/config-svc/pkg/metacfg"

type Cluster struct {
	configRevision int64
	currentNodes   []string
	cfg            metacfg.ClusterConfig
}
