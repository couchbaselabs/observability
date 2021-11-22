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

# Simple helper script to retrieve the associated Secret for a CAO-managed cluster.
set -eu

NAMESPACE=${NAMESPACE:-default}

# General process is:
# Loop for every cluster
# 1. Find CRD for cluster
# 2. Get the spec.cluster.security.adminSecret field that names the secret to use for that cluster
# 3. Get credentials from secret
# 4. Add to cluster manager
# End loop
for CLUSTER in $(kubectl get -n "$NAMESPACE" couchbaseclusters.couchbase.com --output=name) ; do
    CLUSTER_SECRET=$(kubectl get -n "$NAMESPACE" "$CLUSTER" --template='{{.spec.security.adminSecret}}')
    echo "Secret $CLUSTER_SECRET for $CLUSTER in namespace $NAMESPACE"
    CB_USERNAME=$(kubectl get secret "$CLUSTER_SECRET" --template='{{.data.username | base64decode }}')
    CB_PASSWORD=$(kubectl get secret "$CLUSTER_SECRET" --template='{{.data.password | base64decode }}')
    echo "Credentials are $CB_USERNAME:$CB_PASSWORD"
done
