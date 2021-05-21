#!/bin/bash
set -x
CONTROL_HOST=${CONTROL_HOST:-couchbase1.compose.local}

# Wait for CB server to start and make the REST API available
echo "Waiting for Couchbase REST API..."
until curl --silent --show-error -X GET -u Administrator:password http://$CONTROL_HOST:8091/pools/default &>/dev/null; do
  echo -n '.'
  sleep 2
done
echo "Couchbase REST API available"

if curl --silent -u Administrator:password http://$CONTROL_HOST:8091/pools/default | grep -q "unknown pool"; then
    echo "New cluster to create"

    # Initialize Node
    curl  -u Administrator:password -v -X POST \
        http://$CONTROL_HOST:8091/nodes/self/controller/settings \
        -d 'path=%2Fopt%2Fcouchbase%2Fvar%2Flib%2Fcouchbase%2Fdata&' \
        -d 'index_path=%2Fopt%2Fcouchbase%2Fvar%2Flib%2Fcouchbase%2Fdata&' \
        -d 'cbas_path=%2Fopt%2Fcouchbase%2Fvar%2Flib%2Fcouchbase%2Fdata&' \
        -d 'eventing_path=%2Fopt%2Fcouchbase%2Fvar%2Flib%2Fcouchbase%2Fdata&'

    # Rename node
    curl  -u Administrator:password -v -X POST http://$CONTROL_HOST:8091/node/controller/rename \
        -d 'hostname=$CONTROL_HOST'

    # Setup Services
    curl  -u Administrator:password -v -X POST http://$CONTROL_HOST:8091/node/controller/setupServices \
        -d 'services=kv%2Cn1ql%2Cindex%2Cfts'

    # Setup Memory Quotas
    curl  -u Administrator:password -v -X POST http://$CONTROL_HOST:8091/pools/default \
        -d 'memoryQuota=256' \
        -d 'indexMemoryQuota=256' \
        -d 'ftsMemoryQuota=256'

    # Setup Administrator username and password
    curl  -u Administrator:password -v -X POST http://$CONTROL_HOST:8091/settings/web \
        -d 'password=password&username=Administrator&port=SAME'

    # Setup Bucket
    curl  -u Administrator:password -v -X POST http://$CONTROL_HOST:8091/pools/default/buckets \
        -d 'flushEnabled=1' \
        -d 'threadsNumber=3' \
        -d 'replicaIndex=0' \
        -d 'replicaNumber=0' \
        -d 'evictionPolicy=valueOnly' \
        -d 'ramQuotaMB=100' \
        -d 'bucketType=membase' \
        -d 'name=default'

    # Add the other nodes
    curl -v -X POST -u Administrator:password \
        http://$CONTROL_HOST:8091/controller/addNode \
        -d 'hostname=http://couchbase2.compose.local' \
        -d 'user=Administrator' \
        -d 'password=password' \
        -d 'services=kv,n1ql,index'

    curl -v -X POST -u Administrator:password \
        http://$CONTROL_HOST:8091/controller/addNode \
        -d 'hostname=http://couchbase3.compose.local' \
        -d 'user=Administrator' \
        -d 'password=password' \
        -d 'services=kv,n1ql,index'

    # Now rebalance all nodes in
    curl -u Administrator:password -v -X POST \
        http://$CONTROL_HOST:8091/controller/rebalance \
        -d 'knownNodes=ns_1%40couchbase1.compose.local%2Cns_1%40couchbase2.compose.local%2Cns_1%40couchbase3.compose.local'
else
    echo "Already set up"
fi
