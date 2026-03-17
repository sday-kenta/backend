package request

type Login struct {
	Identifier string `json:"identifier" validate:"required" example:"user123"`
	Password   string `json:"password" validate:"required" example:"qwerty123"`
}

