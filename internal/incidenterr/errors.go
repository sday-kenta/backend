package incidenterr

import "errors"

var (
	ErrNotFound           = errors.New("incident not found")
	ErrPhotoNotFound      = errors.New("incident photo not found")
	ErrForbidden          = errors.New("access denied")
	ErrInvalidStatus      = errors.New("invalid incident status")
	ErrInvalidCoordinates = errors.New("both latitude and longitude must be provided")
	ErrLocationRequired   = errors.New("address or coordinates are required")
	ErrCategoryNotFound   = errors.New("category not found")
	ErrDocumentEmailEmpty = errors.New("email is required")
)
