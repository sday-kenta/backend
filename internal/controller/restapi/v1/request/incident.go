package request

// CreateIncident creates an incident/message.
type CreateIncident struct {
	CategoryID     int      `json:"category_id" validate:"required" example:"1"`
	Title          string   `json:"title" validate:"required,max=255" example:"Неправильная парковка во дворе"`
	Description    string   `json:"description" validate:"required,max=600" example:"Автомобиль припаркован на газоне и мешает проходу."`
	Status         string   `json:"status" validate:"omitempty,oneof=draft review published" example:"review"`
	DepartmentName string   `json:"department_name,omitempty" example:"ГИБДД"`
	City           string   `json:"city,omitempty" example:"Самара"`
	Street         string   `json:"street,omitempty" example:"проспект Ленина"`
	House          string   `json:"house,omitempty" example:"1"`
	AddressText    string   `json:"address_text,omitempty" example:"Самара, проспект Ленина, 1"`
	Latitude       *float64 `json:"latitude,omitempty" example:"53.2051714"`
	Longitude      *float64 `json:"longitude,omitempty" example:"50.1334676"`
}

// UpdateIncident updates incident/message fields.
type UpdateIncident struct {
	CategoryID     *int     `json:"category_id,omitempty" example:"1"`
	Title          *string  `json:"title,omitempty" validate:"omitempty,max=255" example:"Неправильная парковка во дворе"`
	Description    *string  `json:"description,omitempty" validate:"omitempty,max=600" example:"Автомобиль припаркован на газоне и мешает проходу."`
	Status         *string  `json:"status,omitempty" validate:"omitempty,oneof=draft review published" example:"review"`
	DepartmentName *string  `json:"department_name,omitempty" example:"ГИБДД"`
	City           *string  `json:"city,omitempty" example:"Самара"`
	Street         *string  `json:"street,omitempty" example:"проспект Ленина"`
	House          *string  `json:"house,omitempty" example:"1"`
	AddressText    *string  `json:"address_text,omitempty" example:"Самара, проспект Ленина, 1"`
	Latitude       *float64 `json:"latitude,omitempty" example:"53.2051714"`
	Longitude      *float64 `json:"longitude,omitempty" example:"50.1334676"`
}

// SendIncidentDocumentEmail requests document delivery by email.
type SendIncidentDocumentEmail struct {
	Email string `json:"email,omitempty" validate:"omitempty,email" example:"user@example.com"`
}
