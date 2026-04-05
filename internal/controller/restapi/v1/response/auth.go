package response

import (
	"time"

	"github.com/sday-kenta/backend/internal/entity"
)

type AuthLogin struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time   `json:"expires_at"`
	User        entity.User `json:"user"`
}
