This example runs up a local KIND cluster (3 worker nodes), deploys Couchbase to it using the CAO and then runs up our monitoring stack.

By default, it gives you the command to handle port-forwarding to the CMOS service on http://localhost:8080/.

An ingress is optionally created (`SKIP_INGRESS=no`) to allow you to access it all via http://localhost/ but this can trigger forwarding issues in some of the CMOS documentation.

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-kubernetes`.

Make sure to build the CMOS container before using the local [`run.sh`](./run.sh) script via a call to `make container` or `make container-oss` as appropriate.

Once you have it running, use the CMOS `Add Cluster` page to add any clusters to monitor but ensure the `Add to Prometheus` option is disabled as we use `kubernetes_sd_config` to auto-discover pods to monitor.
