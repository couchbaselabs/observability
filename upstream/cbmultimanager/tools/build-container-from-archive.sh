#!/usr/bin/env bash

#
# Copyright (C) 2022 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#
if [ $# -eq 0 ]; then
    echo "No image file supplied."
    echo "Usage: build-container-from-archive.sh path/to/image.tgz"
    exit 1
fi

file=$1
arch=$2

# Use a specific known version by default, but allow overriding at runtime.
# This is necessary to ensure we can download notices.txt in the Dockerfile.
VERSION=${VERSION:-0.2.0}
BLD_NUM=${BLD_NUM:-278}
TAG=${TAG:-}

function build_single_image() {
    local artifacts_path=$1
    local product_name=$2
    local arch=$3
    local tag=""
    if [ -z "$TAG" ]; then
        tag="couchbase/${product_name#couchbase-}:${VERSION}-${BLD_NUM}"
    else
        tag="couchbase/${product_name#couchbase-}:$TAG"
    fi
    echo "Test-building Docker image $tag..."
    set -x
    docker build --build-arg PROD_VERSION="$VERSION" --build-arg PROD_BUILD="$BLD_NUM" --build-arg arch="$arch" -t "$tag" "$artifacts_path"
    set +x
}

tmpdir=$(mktemp -d)
tar -C "$tmpdir" -zxvf "$file"

if [ -f "$tmpdir/Dockerfile" ]; then
    product_name=$(basename "$file")
    product_name=${product_name%-image*}
    build_single_image "$tmpdir" "$product_name"
else
    for product_path in "$tmpdir"/*; do
        build_single_image "$product_path" "$(basename "$product_path")" "$arch"
    done
fi
