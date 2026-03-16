package usererr

import "errors"

var (
	ErrNotFound        = errors.New("user not found")
	ErrDuplicateLogin  = errors.New("login already exists")
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrDuplicatePhone  = errors.New("phone already exists")
	ErrInvalidRole     = errors.New("invalid role")
)
