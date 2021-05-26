package values

import "errors"

// ErrNotFound is a generic error for when we cannot get a resource.
var ErrNotFound = errors.New("resource not found")
