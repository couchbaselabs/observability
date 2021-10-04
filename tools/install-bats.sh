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
set -xueo pipefail

# Simple script to install all the BATS helpers.
# BATS itself should be installed via the installation methods documented: https://bats-core.readthedocs.io/en/stable/installation.html

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

export BATS_ROOT=${BATS_ROOT:-$SCRIPT_DIR/bats}
export BATS_FILE_ROOT=$BATS_ROOT/lib/bats-file
export BATS_SUPPORT_ROOT=$BATS_ROOT/lib/bats-support
export BATS_ASSERT_ROOT=$BATS_ROOT/lib/bats-assert
export BATS_DETIK_ROOT=$BATS_ROOT/lib/bats-detik
rm -rf  "${BATS_ROOT}"
mkdir -p "${BATS_ROOT}/lib"

BATS_ASSERT_VERSION=${BATS_ASSERT_VERSION:-2.0.0}
BATS_SUPPORT_VERSION=${BATS_SUPPORT_VERSION:-0.3.0}
BATS_FILE_VERSION=${BATS_FILE_VERSION:-0.3.0}
BATS_DETIK_VERSION=${BATS_DETIK_VERSION:-1.0.0}

DOWNLOAD_TEMP_DIR=$(mktemp -d)

# Install BATS helpers using specified versions
pushd "${DOWNLOAD_TEMP_DIR}"
    curl -LO "https://github.com/bats-core/bats-assert/archive/refs/tags/v$BATS_ASSERT_VERSION.zip"
    unzip -q "v$BATS_ASSERT_VERSION.zip"
    mv -f "${DOWNLOAD_TEMP_DIR}/bats-assert-$BATS_ASSERT_VERSION" "${BATS_ASSERT_ROOT}"
    rm -f "v$BATS_ASSERT_VERSION.zip"

    curl -LO "https://github.com/bats-core/bats-support/archive/refs/tags/v$BATS_SUPPORT_VERSION.zip"
    unzip -q "v$BATS_SUPPORT_VERSION.zip"
    mv -f "${DOWNLOAD_TEMP_DIR}/bats-support-$BATS_SUPPORT_VERSION" "${BATS_SUPPORT_ROOT}"
    rm -f "v$BATS_SUPPORT_VERSION.zip"

    curl -LO "https://github.com/bats-core/bats-file/archive/refs/tags/v$BATS_FILE_VERSION.zip"
    unzip -q "v$BATS_FILE_VERSION.zip"
    mv -f "${DOWNLOAD_TEMP_DIR}/bats-file-$BATS_FILE_VERSION" "${BATS_FILE_ROOT}"
    rm -f "v$BATS_FILE_VERSION.zip"

    curl -LO "https://github.com/bats-core/bats-detik/archive/refs/tags/v$BATS_DETIK_VERSION.zip"
    unzip -q "v$BATS_DETIK_VERSION.zip"
    mv -f "${DOWNLOAD_TEMP_DIR}/bats-detik-$BATS_DETIK_VERSION/lib" "${BATS_DETIK_ROOT}"
    rm -f "v$BATS_DETIK_VERSION.zip"
popd
rm -rf "${DOWNLOAD_TEMP_DIR}"
