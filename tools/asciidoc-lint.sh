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

# Simple linting of documentation helper script.
set -eu
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# FILTER allows us to pre-process the source and discard irrelevant parts.
FILTER="sed"
# Ignore source code blocks (contain arbitrary non-English configuration).
FILTER="${FILTER} -e /^----$/,/^----$/d"
# Ignore asciidoc ifdef blocks (contain arbitrary non-English configuration).
FILTER="${FILTER} -e /^ifdef/,/^endif/d"
# Ignore inline literals and attributes (contain arbitrary non-English configuration).
FILTER="${FILTER} -e s/\`[^\`]*\`//g"
FILTER="${FILTER} -e s/{[^}]*}//g"
# Ignore definition lists (contain arbitrary non-English configuration).
FILTER="${FILTER} -e s/^.*::$//g"
# Ignore hyperlinks (not our problem).
FILTER="${FILTER} -e s/http:[[^\[]//g"
FILTER="${FILTER} -e s/https:[[^\[]//g"
# Ignore cross-reference file names (not user-visisble, however this may affect SEO).
FILTER="${FILTER} -e s/xref:[^\[]*//g"
# Ignore image file names (not user-visisble, however this may affect SEO).
FILTER="${FILTER} -e s/image:[^\[]*//g"
# Ignore asciidoc directives.
FILTER="${FILTER} -e s/^:.*$//g"
# Ignore links anchors.
FILTER="${FILTER} -e s/<<[^,]*//g"
# Ignore includes.
FILTER="${FILTER} -e s/^include::.*$//g"
# Ignore toc commands.
FILTER="${FILTER} -e s/^toc::.*$//g"
# Ignore anchors formatting directives etc.
FILTER="${FILTER} -e s/^\[.*$//g"
# Ignore inline ui-macros.
FILTER="${FILTER} -e s/btn:\[[^\]*\]//g"
FILTER="${FILTER} -e s/kbd:\[[^\]*\]//g"

# CHECK_ARGS is the spell checking command to run.
CHECK_ARGS="-l en_US --home-dir=$SCRIPT_DIR/../"

# Before we spell check, ensure that the docs are generated...
make docs -C "$SCRIPT_DIR/../"

GIT_STATUS="$(git status --short)"
if [[ ${GIT_STATUS} != "" ]]; then
  echo "Documentation has not been generated and committed:"
  echo "${GIT_STATUS}"
  git diff
  echo
  echo "Run 'make docs' and commit the result."
  exit 1
fi

# For each asciidoc we find in the documentation, filter out the
# stuff that isn't real text and spell check the remainder.  For now
# only look in the top-level directory, anything outside of it is
# considered auto-generated and exempt.
FAIL=""
while IFS= read -r -d '' DOCUMENT; do
  # shellcheck disable=SC2086
  MISTAKES=$(${FILTER} "${DOCUMENT}" | aspell list ${CHECK_ARGS} | sort | uniq)
  if [[ ${MISTAKES} != "" ]]; then
    echo "Spell check for file ${DOCUMENT} failed:"
    # shellcheck disable=SC2001
    echo "${MISTAKES}" | sed 's/^/    /g'
    FAIL="yes"
  fi
done < <(find docs/ -type f -name '*.adoc' -print0)

# Any failures need to be fixed up, and we need to report and error for CI.
if [[ ${FAIL} != "" ]]; then
  echo "Documentation contains errors"
  echo "    run 'aspell check ${CHECK_ARGS}' to fix typos or add to .aspell.en.pws"
  exit 1
fi
