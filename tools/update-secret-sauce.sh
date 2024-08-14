#!/bin/bash
set -euo pipefail
#
# Copyright (C) 2022 Couchbase, Inc.
#
# Use of this software is subject to the Couchbase Inc. License Agreement
# which may be found at https://www.couchbase.com/LA03012021.
#

# This script temporarily clones the support-secret-sauce repository, extracts the data we need from it, and places
# copies of it in the cbmultimanager repository.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

function print_help() {
  cat <<EOF
update-secret-sauce.sh: updates data derived from https://github.com/couchbaselabs/support-secret-sauce

Parameters:
  -l <path>, --local <path>           Use a local copy of the support-secret-sauce data instead of cloning from Git
  -r <url>, --remote <url>            Use the specified Git URL instead of the default (ignored if -l is used)
  -b <branch>, --branch <branch>      Clone the specified branch or tag instead of main (ignored if -l is used)
  -h, --help                          Print this message and exit
EOF
  exit 0
}

REMOTE="git@github.com:couchbaselabs/support-secret-sauce.git"
CLONE=1
BRANCH="main"
DEST_PATH=""

while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      print_help
      ;;
    -l|--local)
      DEST_PATH="$2"
      CLONE=0
      shift
      shift
      ;;
    -r|--remote)
      REMOTE="$2"
      shift
      shift
      ;;
    -b|--branch)
      BRANCH="$2"
      shift
      shift
      ;;
  esac
done

if [ "$CLONE" -eq 1 ]; then
  DEST_PATH=$(mktemp -d)
  echo "Cloning support-secret-sauce @ $BRANCH to $DEST_PATH..."
  git clone -b "$BRANCH" "$REMOTE" "$DEST_PATH"
else
  echo "Using secret sauce from $DEST_PATH"
fi

echo "Running transformers..."

echo "cbvers..."
# Compact it so that the final binary is smaller
jq -c '[.versions[] | select(.generallyAvailable) | del(.extra) | .os = ((.os // []) | map(del(.extra) | del(.checkerLevel)))]' "$DEST_PATH/versions/cbvers.json" > "$SCRIPT_DIR/../cluster-monitor/pkg/values/versions.json"

if [ "$CLONE" -eq 1 ]; then
  echo "Cleaning up..."
  rm -rf "$DEST_PATH"
fi
