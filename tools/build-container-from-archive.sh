#!/usr/bin/env bash

if [ $# -eq 0 ]; then
    echo "No image file supplied."
    echo "Usage: build-container-from-archive.sh path/to/image.tgz"
    exit 1
fi

file=$1

if [ $# -eq 2 ]; then
    tag=$2
else
    file_basename=$(basename "$file")
    tag=${file_basename%%.tgz}
    tag=${tag%%.tar.gz}
    tag=$(echo "$tag" | sed -E 's/^couchbase-(.+)-image_(.+)$/couchbase\/\1:\2/')
fi


echo "Test-building Docker image $tag..."
tmpdir=$(mktemp -d)
tar -C "$tmpdir" -zxvf "$file"
docker build -t "$tag" --build-arg PROD_VERSION=0.2.0 --build-arg PROD_BUILD=278 "$tmpdir"
