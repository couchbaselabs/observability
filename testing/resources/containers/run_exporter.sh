#!/usr/bin/env bash

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

# Start the couchbase-exporter.
# Both this and CBS will be started by runsvdir-start concurrently, so it is possible that
# the exporter starts before CBS starts. As of https://github.com/couchbase/couchbase-exporter/commit/61914d56c15580b59910cca97eaedaed4aafbcff
# this should be safe.
exec /opt/couchbase-exporter
