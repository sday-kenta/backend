package categoryerr

import "errors"

var (
	ErrNotFound = errors.New("category not found")
)
