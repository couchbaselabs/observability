#!/bin/bash
#
# Copyright (C) 2021 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#

# Simple script to check all files have the appropriate copyright, will fail and list them if not.
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

exitCode=0
while IFS= read -r -d '' SOURCE
do
    if ! head "${SOURCE}" | grep -qie 'Copyright (C) [[:digit:]][[:digit:]][[:digit:]][[:digit:]] Couchbase, Inc.'; then
        echo "FAILED: Missing copyright: .${SOURCE##"$SCRIPT_DIR"/..}"
        exitCode=1
    fi
    if ! head "${SOURCE}" | grep -qie 'Use of this software is subject to the Couchbase Inc. License Agreement'; then
        echo "FAILED: Missing license: .${SOURCE##"$SCRIPT_DIR"/..}"
        exitCode=1
    fi
done < <(find "${SCRIPT_DIR}/.." -type f \( -name '*.go' -o -name '*.sh' \) ! -path "${SCRIPT_DIR}/../upstream/*" ! -path '*mocks*' -print0)
# Make sure we prune out any local Go installation directory

exit $exitCode