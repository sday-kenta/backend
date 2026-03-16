package entity

import "errors"

var (
	ErrAddressNotFound    = errors.New("address not found")
	ErrOutOfAllowedZone   = errors.New("project does not work in this area yet")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidSearchQuery = errors.New("search query must contain at least 3 characters")
	ErrZoneNotFound       = errors.New("zone not found")
)

// Address is the internal map/address entity used across controller, use case and repositories.
// JSON tags are intentionally kept here because handlers currently serialize entity.Address directly.
// When dedicated response DTO mapping is introduced, these tags can be removed from the domain layer.
type Address struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	City        string  `json:"city,omitempty"`
	Road        string  `json:"road,omitempty"`
	HouseNumber string  `json:"house_number,omitempty"`
	FullAddress string  `json:"full_address"`
}

// Zone describes a supported project area and its human-readable city label.
type Zone struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}
