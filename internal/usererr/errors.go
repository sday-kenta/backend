package usererr

import "errors"

var (
	ErrNotFound        = errors.New("user not found")
	ErrDuplicateLogin  = errors.New("login already exists")
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrDuplicatePhone  = errors.New("phone already exists")
	ErrInvalidRole     = errors.New("invalid role")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBlocked        = errors.New("user is blocked")
	ErrInvalidCode        = errors.New("invalid code")
	ErrCodeExpired   = errors.New("code expired")
	ErrInvalidPhone      = errors.New("phone must be in format +7 (XXX) XXX-XX-XX (11 digits only)")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters")
	ErrEmailNotVerified  = errors.New("email not verified")
)
