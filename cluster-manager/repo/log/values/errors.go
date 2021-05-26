package values

import "errors"

var (
	ErrNotInLine           = errors.New("event not in this line")
	ErrNotFullLine         = errors.New("part of line is missing; getting next section of line")
	ErrRegexpMissingFields = errors.New("not all regexp capture groups found")
	ErrAlreadyInLog        = errors.New("line already in events log")
)
