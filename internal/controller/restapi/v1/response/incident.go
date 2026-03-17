package response

import "time"

// IncidentPhoto is a public API representation of incident photo metadata.
type IncidentPhoto struct {
	ID          int64     `json:"id" example:"10"`
	FileURL     string    `json:"file_url" example:"http://localhost:9000/avatars/incidents/1/photo-1.jpg"`
	ContentType string    `json:"content_type,omitempty" example:"image/jpeg"`
	SizeBytes   int64     `json:"size_bytes" example:"245678"`
	SortOrder   int       `json:"sort_order" example:"0"`
	CreatedAt   time.Time `json:"created_at"`
}

// Incident is a public API representation of an incident without reporter PII.
type Incident struct {
	ID             int64           `json:"id" example:"1"`
	UserID         int64           `json:"user_id" example:"2"`
	CategoryID     int             `json:"category_id" example:"1"`
	CategoryTitle  string          `json:"category_title" example:"Нарушение правил парковки"`
	Title          string          `json:"title" example:"Неправильная парковка во дворе"`
	Description    string          `json:"description" example:"Автомобиль припаркован на газоне и мешает проходу."`
	Status         string          `json:"status" example:"published"`
	DepartmentName string          `json:"department_name" example:"ГИБДД"`
	City           string          `json:"city,omitempty" example:"Самара"`
	Street         string          `json:"street,omitempty" example:"проспект Ленина"`
	House          string          `json:"house,omitempty" example:"1"`
	AddressText    string          `json:"address_text" example:"Самара, проспект Ленина, 1"`
	Latitude       float64         `json:"latitude" example:"53.2051714"`
	Longitude      float64         `json:"longitude" example:"50.1334676"`
	Photos         []IncidentPhoto `json:"photos,omitempty"`
	PublishedAt    *time.Time      `json:"published_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// MessageResponse is a generic success response.
type MessageResponse struct {
	Message string `json:"message" example:"ok"`
}
