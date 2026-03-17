package request

type SendPasswordResetCode struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

