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
set -eu

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Run envsubst on all test files that might need it
while IFS= read -r -d '' INPUT_FILE; do
    OUTPUT_FILE=${INPUT_FILE%%-template.yaml}.yaml
    echo "[ENTRYPOINT]: Substitute template ${INPUT_FILE} --> ${OUTPUT_FILE}"
    # Make sure to leave alone anything that is not a defined environment variable
    envsubst "$(env | cut -d= -f1 | sed -e 's/^/$/')"  < "${INPUT_FILE}" > "${OUTPUT_FILE}"
    #cat "${OUTPUT_FILE}"
done < <(find "${SCRIPT_DIR}/" -type f -name '*-template.yaml' -print0)

if [[ $# -gt 0 ]]; then
    echo "[ENTRYPOINT] Running custom: $*"
    exec "$@"
else
    exec bats --timing --recursive "${SCRIPT_DIR}/"
fi