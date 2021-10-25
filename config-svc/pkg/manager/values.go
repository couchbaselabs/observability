// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import "github.com/couchbaselabs/observability/config-svc/pkg/metacfg"

type clusterState struct {
	uuid         string
	currentNodes []string
	cfg          *metacfg.ClusterConfig
}

// ClusterInfo is a combination of the configured metadata from metacfg, as well as the current nodes state.
type ClusterInfo struct {
	UUID            string
	Nodes           []string
	Metadata        map[string]string
	CouchbaseConfig metacfg.CouchbaseConfig
	MetricsConfig   metacfg.MetricsConfig
}

type ClusterInfoListener chan<- ClusterInfo
