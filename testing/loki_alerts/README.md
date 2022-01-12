# Loki alert unit testing

This directory contains scripts and helpers for testing our Loki alerting rules.

**NB**: As a simplification, these tests only test that the alert's LogQL query matches. It does not check the alert labels.

## Adding a new test

1. Create a new directory called `<group name>/<rule name>` in this folder, where `<group name>` and `<rule name>` come from the YAML rules definition file (in microlith/loki/alerting/couchbase).
2. Place the log files that should trigger that rule in that directory.
    * These must be named as they would be on a running Couchbase Server (so if you're using a cbcollect make sure to trim off the `ns_server.` prefix)
    * These MUST NOT contain any customer data! Where possible, use logs from your own reproduction of the issue.
3. To verify, run `./run_single_test.sh <group name>/<rule name>`.
