#!/usr/bin/env bash
set -e

trap "exit" INT TERM
trap 'kill $(jobs -p)' EXIT

for i in /entrypoints/*; do
    EXE_NAME=${i##/entrypoints/}
    UPPERCASE=${EXE_NAME^^}
    DISABLE_VAR=DISABLE_${UPPERCASE%%.*}
    # Set DISABLE_XXX to skip running
    if [[ -v "${DISABLE_VAR}" ]]; then
        echo "[ENTRYPOINT] Disabled as ${DISABLE_VAR} set (value ignored): $i"
    elif [[ -x "$i" ]]; then
        # See https://github.com/hilbix/speedtests for log name pre-pending info
        echo "[ENTRYPOINT] Running: $i"
        tini -- "$i" "$@" 2>&1 | awk '{ print "['"${EXE_NAME}"']" $0 }' &
    else
        echo "[ENTRYPOINT] Skipping non-executable: $i"
    fi
done

wait -n