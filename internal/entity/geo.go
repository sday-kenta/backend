package entity

import (
	"errors"
	"strings"
)

var (
	ErrAddressNotFound    = errors.New("address not found")
	ErrOutOfAllowedZone   = errors.New("project does not work in this area yet")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidSearchQuery = errors.New("search query must contain at least 3 characters")
)

// Address is the internal map/address entity used across controller, use case and repositories.
type Address struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	City        string  `json:"city,omitempty"`
	Road        string  `json:"road,omitempty"`
	HouseNumber string  `json:"house_number,omitempty"`
	FullAddress string  `json:"full_address"`
}

// NominatimPlace models json/jsonv2 output of Nominatim /search and /reverse.
type NominatimPlace struct {
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	DisplayName string           `json:"display_name"`
	Address     NominatimAddress `json:"address"`
	Error       string           `json:"error,omitempty"`
}

// NominatimAddress contains the address breakdown from Nominatim.
// The API returns categories according to OSM tags, so multiple fields must be checked.
type NominatimAddress struct {
	HouseNumber string `json:"house_number"`
	Road        string `json:"road"`
	Pedestrian  string `json:"pedestrian"`
	Footway     string `json:"footway"`
	Path        string `json:"path"`
	Street      string `json:"street"`

	City         string `json:"city"`
	Town         string `json:"town"`
	Village      string `json:"village"`
	Hamlet       string `json:"hamlet"`
	Municipality string `json:"municipality"`
	County       string `json:"county"`
	State        string `json:"state"`
	Country      string `json:"country"`
}

func (n NominatimAddress) GetCity() string {
	for _, candidate := range []string{n.City, n.Town, n.Village, n.Hamlet, n.Municipality, n.County, n.State} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

func (n NominatimAddress) GetRoad() string {
	for _, candidate := range []string{n.Road, n.Pedestrian, n.Footway, n.Path, n.Street} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}

	return ""
}
