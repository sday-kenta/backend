package request

type CreateUser struct {
	Login       string `json:"login"        validate:"required"  example:"user123"`
	Email       string `json:"email"        validate:"required,email"  example:"user@example.com"`
	Password    string `json:"password"     validate:"required,min=6"  example:"qwerty123"`
	LastName    string `json:"last_name"    validate:"required"  example:"Иванов"`
	FirstName   string `json:"first_name"   validate:"required"  example:"Иван"`
	MiddleName  string `json:"middle_name"  example:"Иванович"`
	Phone       string `json:"phone"        validate:"required"  example:"+79991234567"`
	City        string `json:"city"         validate:"required"  example:"Москва"`
	Street      string `json:"street"       validate:"required"  example:"Тверская"`
	House       string `json:"house"        validate:"required"  example:"1"`
	Apartment   string `json:"apartment"    example:"10"`
	IsBlocked   bool   `json:"is_blocked" example:"false"`
	Role        string `json:"role" validate:"required,oneof=user admin premium" example:"user"`
}

type UpdateUser struct {
	Login      string `json:"login"        validate:"required"  example:"user123"`
	Email      string `json:"email"        validate:"required,email"  example:"user@example.com"`
	LastName   string `json:"last_name"    validate:"required"  example:"Иванов"`
	FirstName  string `json:"first_name"   validate:"required"  example:"Иван"`
	MiddleName string `json:"middle_name"  example:"Иванович"`
	Phone      string `json:"phone"        validate:"required"  example:"+79991234567"`
	City       string `json:"city"         validate:"required"  example:"Москва"`
	Street     string `json:"street"       validate:"required"  example:"Тверская"`
	House      string `json:"house"        validate:"required"  example:"1"`
	Apartment  string `json:"apartment"    example:"10"`
	IsBlocked bool   `json:"is_blocked" example:"false"`
	Role      string `json:"role" validate:"required,oneof=user admin premium" example:"user"`
}

