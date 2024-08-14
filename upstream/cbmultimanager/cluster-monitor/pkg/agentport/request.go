// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package agentport

import (
	"context"
	"time"
)

// RequestMethod is a common HTTP call.
type RequestMethod func(*AgentPort, context.Context, *Request) error

// Request is a one time request against the Client API.
type Request struct {
	// method is the API routine to use, analogous to GET, POST etc.
	method RequestMethod

	// path is the HTTP path to call.
	Path string

	// body is the request body.
	Body []byte

	// result is the unmarshalled response body.
	Result interface{}

	// timeout is the duration to allow retries for.
	timeout time.Duration

	// err records any errors during the request build process.
	err error

	// whether failures in this request should result in attempts to reinit
	// the agent
	revive bool
}

// NewRequest is a terse way of creating an API request.
func NewRequest(method RequestMethod, path string, body []byte, result interface{}) *Request {
	return &Request{
		method: method,
		Path:   path,
		Body:   body,
		Result: result,
	}
}

func (r *Request) WithTimeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

// NewRequestError is how to return an error in a builder chain.  It will be picked
// up by Execute and reported then.
func NewRequestError(err error) *Request {
	return &Request{
		err: err,
	}
}

func (r *Request) Execute(c *AgentPort) error {
	if r.err != nil {
		return r.err
	}
	if r.timeout != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		err := r.method(c, ctx, r)

		return err
	}

	return r.method(c, context.Background(), r)
}

func (r *Request) WithRevive() *Request {
	r.revive = true
	return r
}
