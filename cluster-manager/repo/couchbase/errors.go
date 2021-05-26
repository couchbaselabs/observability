package couchbase

import "fmt"

type AuthError struct {
	Authentication bool
	err            error
}

func (e AuthError) Error() string {
	return fmt.Sprintf("invalid auth: %v", e.err)
}

func (e AuthError) Unwrap() error {
	return e.err
}
