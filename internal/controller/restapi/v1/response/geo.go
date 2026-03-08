package response

type GeoAddress struct {
	Lat         float64 `json:"lat" example:"53.2051714"`
	Lon         float64 `json:"lon" example:"50.1334676"`
	City        string  `json:"city" example:"Самара"`
	Road        string  `json:"road" example:"проспект Ленина"`
	HouseNumber string  `json:"house_number,omitempty" example:"1"`
	FullAddress string  `json:"full_address" example:"1, проспект Ленина, Октябрьский район, Самара, городской округ Самара, Самарская область, Приволжский федеральный округ, 443110, Россия"`
}

type SearchAddressResponse struct {
	Status string       `json:"status" example:"success"`
	Data   []GeoAddress `json:"data"`
}

type ReverseGeocodeResponse struct {
	Status string     `json:"status" example:"success"`
	Data   GeoAddress `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"project does not work in this area yet"`
}
