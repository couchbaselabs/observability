#!/usr/bin/env bash
set -e
for i in /entrypoints/*; do
    if [[ -x "$i" ]]; then
        LOGFILE=${LOGS_DIR:-/logs}/${i##/entrypoints/}.log
        echo "Running: $i > $LOGFILE"
        tini -- "$i" "$@" &> "${LOGFILE}" &
    else
        echo "Skipping non-executable: $i"
    fi
done

wait -n