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

package couchbase

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/couchbase/tools-common/cbvalue"
	"github.com/labstack/echo/v4"
)

type Node struct {
	Hostname           string          `json:"hostname"`
	Version            cbvalue.Version `json:"version"`
	AlternateAddresses map[string]struct {
		Hostname string         `json:"hostname"`
		Ports    map[string]int `json:"ports"`
	} `json:"alternateAddresses,omitempty"`
}

type PoolsDefault struct {
	ClusterName string `json:"clusterName"`
	Nodes       []Node `json:"nodes"`
}

func (n Node) ResolveHostPort(secure bool) (string, int, error) {
	hostname := n.Hostname
	mgmtPort := 8091

	if externalAddr, ok := n.AlternateAddresses["external"]; ok {
		hostname = externalAddr.Hostname
		mgmtPort = externalAddr.Ports["mgmt"]
		if sslPort, ok := externalAddr.Ports["mgmtSSL"]; secure && ok {
			mgmtPort = sslPort
		}
	} else {
		host, port, err := net.SplitHostPort(hostname)
		if err == nil {
			hostname = host
			mgmtPort, err = strconv.Atoi(port)
			if err != nil {
				return "", 0, fmt.Errorf("failed to parse CB hostname port: %w", err)
			}
		}
	}
	return hostname, mgmtPort, nil
}

func FetchCouchbaseClusterInfo(hostname string, port int, secure bool, username, password string) (*PoolsDefault,
	error) {
	// First, fetch the list of targets from CBS
	scheme := "http"
	if secure {
		scheme = "https"
	}
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s://%s:%d/pools/default", scheme, hostname, port),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}
	req.SetBasicAuth(username, password)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Couchbase Server: %s", err.Error())
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Couchbase body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Couchbase Server returned non-OK code %d: %s",
			res.StatusCode, string(body)))
	}

	var cluster PoolsDefault
	if err := json.Unmarshal(body, &cluster); err != nil {
		return nil, fmt.Errorf("failed to parse Couchbase body: %w", err)
	}
	return &cluster, nil
}
