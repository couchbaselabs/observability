This example runs up a local KIND cluster (3 worker nodes), deploys Couchbase to it using the CAO and then runs up our monitoring stack.

An ingress is created to allow you to access it all via http://localhost/

Make sure to build the CMOS container before using the local [`run.sh`](./run.sh) script via a call to `make container` or `make container-oss` as appropriate.
