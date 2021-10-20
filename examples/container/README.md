An example of using the microlith image locally.

To run a full stack use the `Makefile` at the top of this repo and just execute the target: `make example-containers`

This uses an SSH mount to access a private git repository during the container build so make sure your SSH keys are set up for git locally and ssh agent is running with them to provide it.

This will spin up a Couchbase cluster (single node) with Prometheus exporter.
It will also build and start the all-in-one observability container and configure it to talk to the cluster automatically.

Add additional clusters by running up a new Couchbase Server image and either attaching it to an existing cluster or creating a new one.

For Linux, make sure to enable the CLI tech preview for docker compose: https://docs.docker.com/compose/cli-command/

It demonstrates how to mount in custom rules and end points to scrape.