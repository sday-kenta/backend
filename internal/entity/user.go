package entity

import "time"

const (
	UserRoleUser    = "user"
	UserRoleAdmin   = "admin"
	UserRolePremium = "premium"
)

// User represents an application user.
// It is used by business logic and maps to the "users" table.
type User struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"` // never expose password hash

	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name,omitempty"`

	Phone     string `json:"phone"`
	City      string `json:"city"`
	Street    string `json:"street"`
	House     string `json:"house"`
	Apartment string `json:"apartment,omitempty"`

	// AvatarURL stores identifier or URL of avatar image (e.g. S3 key).
	AvatarURL string `json:"avatar_url,omitempty"`

	IsBlocked bool   `json:"is_blocked"`
	Role      string `json:"role"` // user, admin, premium

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
