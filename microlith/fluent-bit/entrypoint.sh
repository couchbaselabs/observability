#!/bin/sh
/fluent-bit/bin/couchbase-watcher &

while true; do
    sleep 5
    /fluent-bit/bin/cb-eventlog
done
