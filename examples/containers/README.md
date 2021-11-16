An example of using the microlith image locally and CMOS is then available via http://localhost:8080.

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-containers`.

This will spin up a Couchbase cluster (single node) and CMOS.
The `Add Cluster` page can be used to add the `db1` host with default credentials of `Administrator:password`.

Add additional clusters by running up a new Couchbase Server image and either attaching it to an existing cluster or creating a new one.

It demonstrates how to mount in custom rules and end points to scrape.

Make sure nothing else is using port 8080 already locally.

Make sure to build the CMOS container before using the local [`run.sh`](./run.sh) script via a call to `make container` or `make container-oss` as appropriate.
