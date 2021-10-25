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

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
)

type poolsDefault struct {
	Name  string `json:"name"`
	Nodes []struct {
		Hostname          string `json:"hostname"`
		Status            string `json:"status"`
		ClusterMembership string `json:"clusterMembership"`
	} `json:"nodes"`
}

// makeRequestToNode performs an HTTP request to the given node and writes the result to out.
// Returns an error if it fails for any reason.
// The URL the request is made to is, roughly, https://{node}:{cfg.ManagementPort}{endpoint}
// (thus the endpoint should have a leading slash).
func (m *ClusterManager) makeRequestToNode(node string, cfg metacfg.CouchbaseConfig, endpoint string,
	out interface{}) error {
	req, err := http.NewRequestWithContext(m.pollingLoopCtx, http.MethodGet,
		fmt.Sprintf("https://%s:%d%s",
			node,
			cfg.ManagementPort,
			endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %w", err)
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)
	client := http.DefaultClient
	if cfg.IgnoreCertificateErrors {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTTP response body: %w", err)
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

// makeRequestToNode performs an HTTP request to the given node and returns a channel of the messages streamed
// by the server. The request URL is formed in the same way as makeRequestToNode (endpoint should have a leading slash).
// resultType should be a pointer to the type of messages you expect to receive from the server (for example,
// `new(someStruct)`). This pointer will not be used for anything other than determining the type to send on the
// returned channel.
//
// The returned channel will receive either pointers to instances of resultType, or instances of error. If an error
// is received, it will always be the last thing sent on the channel before it is closed.
//
// Note that unexpected EOFs are not treated as errors, as these may simply indicate the server closing the streaming
// connection.
func (m *ClusterManager) makeStreamingRequestToNode(ctx context.Context, node string,
	cfg metacfg.CouchbaseConfig, endpoint string, resultType interface{}) (<-chan interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://%s:%d%s",
			node,
			cfg.ManagementPort,
			endpoint), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %w", err)
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)
	client := http.DefaultClient
	if cfg.IgnoreCertificateErrors {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req) //nolint:bodyclose // we do it in the goroutine below
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	result := make(chan interface{})

	// We need to use reflection here because, unlike in the simple request-response case, here we may need to
	// return N instances of resultType, so we can't just pass one of them to json.Unmarshal
	typ := reflect.TypeOf(resultType)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	go func() {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		for {
			pointer := reflect.New(typ)
			err := decoder.Decode(pointer.Interface())
			if err != nil {
				// an unexpected EOF can mean the server just closed the connection, nothing to panic about
				// still log it, just in case it becomes more frequent
				if errors.Is(err, io.ErrUnexpectedEOF) {
					m.logger.Debugw("unexpected EOF in streaming connection (could just be server close)",
						"node", node)
				} else {
					result <- fmt.Errorf("while processing JSON: %w", err)
				}
				close(result)
				return
			}
			result <- pointer.Interface()
		}
	}()

	return result, nil
}
