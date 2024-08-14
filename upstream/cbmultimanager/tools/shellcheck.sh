#!/bin/bash
# Copyright (C) 2021 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# Find all shell scripts that are not part of the Go local directory used during build.
# Run Shellcheck on them.
# Pruning is a lot more performant as it does not descend into the directory.
# Note we cannot do an exec without some horrible mess to deal with the exit code collection.
# find "${SCRIPT_DIR}/../" \
#     -type d -path "*/go" -prune -o \
#     -type f \( -name '*.sh' -o -name '*.bash' \) -exec sh -c 'echo Shellcheck "$1"; docker run -i --rm koalaman/shellcheck:stable - < "$1"' sh {} \;
exitCode=0
while IFS= read -r -d '' file; do
    echo "Shellcheck: .${file##"$SCRIPT_DIR"/..}"
    if ! docker run -i --rm koalaman/shellcheck:stable - < "$file"; then
        exitCode=1
    fi
done < <(find "${SCRIPT_DIR}/.." -type f -name '*.sh' ! \( -path "${SCRIPT_DIR}/../upstream/*" \) -print0)

exit $exitCode
