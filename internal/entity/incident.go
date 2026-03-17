package entity

import "time"

const (
	IncidentStatusDraft     = "draft"
	IncidentStatusPublished = "published"
)

// Incident is the main domain entity for a user-created incident/message.
type Incident struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	CategoryID    int    `json:"category_id"`
	CategoryTitle string `json:"category_title,omitempty"`

	Title          string `json:"title"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	DepartmentName string `json:"department_name,omitempty"`

	City        string  `json:"city,omitempty"`
	Street      string  `json:"street,omitempty"`
	House       string  `json:"house,omitempty"`
	AddressText string  `json:"address_text"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`

	// Reporter snapshot is stored to keep generated documents stable even if
	// the user updates profile data later.
	ReporterFullName string `json:"-"`
	ReporterEmail    string `json:"-"`
	ReporterPhone    string `json:"-"`
	ReporterAddress  string `json:"-"`

	Photos      []IncidentPhoto `json:"photos,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// IncidentPhoto stores incident photo metadata.
type IncidentPhoto struct {
	ID          int64     `json:"id"`
	IncidentID  int64     `json:"incident_id"`
	FileKey     string    `json:"-"`
	FileURL     string    `json:"file_url"`
	ContentType string    `json:"content_type,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// IncidentFilter is used for incidents listing.
type IncidentFilter struct {
	UserID        *int64
	CategoryID    *int
	Status        *string
	OnlyPublished bool
}

// CreateIncidentInput contains data required for incident creation.
type CreateIncidentInput struct {
	CategoryID     int
	Title          string
	Description    string
	Status         string
	DepartmentName string
	City           string
	Street         string
	House          string
	AddressText    string
	Latitude       *float64
	Longitude      *float64
}

// UpdateIncidentInput contains mutable incident fields.
type UpdateIncidentInput struct {
	CategoryID     *int
	Title          *string
	Description    *string
	Status         *string
	DepartmentName *string
	City           *string
	Street         *string
	House          *string
	AddressText    *string
	Latitude       *float64
	Longitude      *float64
}

// IncidentDocument is a rendered document derived from an incident.
type IncidentDocument struct {
	FileName    string
	ContentType string
	Subject     string
	BodyHTML    string
}
