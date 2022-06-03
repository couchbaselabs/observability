#!/bin/bash
# Copyright 2021 Couchbase, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file  except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the  License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -exuo pipefail

if [ $# -eq 0 ]; then
    echo "No image file supplied."
    echo "Usage: build-container-from-archive.sh path/to/image.tgz"
    exit 1
fi

file=$1
arch=$2

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
    docker buildx build --load --build-arg PROD_VERSION="$VERSION" --build-arg PROD_BUILD="$BLD_NUM" -t "$tag" "$artifacts_path"
}

tmpdir=$(mktemp -d)
tar -C "$tmpdir" -zxvf "$file"

if [ -f "$tmpdir/Dockerfile" ]; then
    product_name=$(basename "$file")
    product_name=${product_name%-image*}
    build_single_image "$tmpdir" "$product_name" "$arch"
else
    for product_path in "$tmpdir"/*; do
        build_single_image "$product_path" "$(basename "$product_path")" "$arch"
    done
fi
