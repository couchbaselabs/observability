#!/usr/bin/env bash
set -e
for i in /entrypoints/*; do
    if [[ -x "$i" ]]; then
        EXE_NAME=${i##/entrypoints/}
        # See https://github.com/hilbix/speedtests for log name pre-pending info
        echo "[ENTRYPOINT] Running: $i"
        tini -- "$i" "$@" 2>&1 | awk '{ print "['"${EXE_NAME}"']" $0 }' &
    else
        echo "[ENTRYPOINT] Skipping non-executable: $i"
    fi
done

wait -n