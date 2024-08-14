#!/usr/bin/env bash

#
# Copyright (C) 2022 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#

set -euo pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

image_files=()

for file in "$SCRIPT_DIR"/../dist/*; do
    file=${file##"$SCRIPT_DIR"/../dist/}
    if [[ "$file" =~ ^.*-image_[0-9]\.[0-9]\.[0-9]-[0-9]+\.(tgz|tar\.gz)$ ]]; then
        echo "Would build Docker image from $file."
        image_files+=("$file")
    elif [[ "$file" =~ [0-9]\.[0-9]\.[0-9]-[0-9]+ ]]; then
        echo "Would archive simple artifact: $file"
    else
        echo "WARNING: Found file in dist not matching pattern: $file"
    fi
done

for file in "${image_files[@]}"; do
    image_name=${file%%.tgz}
    image_name=${image_name%%.tar.gz}
    image_name=$(echo "$image_name" | sed -E 's/^couchbase-(.+)-image_(.+)$/couchbase\/\1:\2/')
    "$SCRIPT_DIR/build-container-from-archive.sh" "$SCRIPT_DIR/../dist/$file" "$image_name"
done
