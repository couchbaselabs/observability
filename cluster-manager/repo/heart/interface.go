package heart

import (
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

type MonitorIFace interface {
	Start(heartBeatFrequency time.Duration)
	Stop()
	HeartBeatCluster(cluster *values.CouchbaseCluster) error
}
