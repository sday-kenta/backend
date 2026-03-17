package request

type SendPasswordResetCode struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

type VerifyPasswordResetCode struct {
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
	Code  string `json:"code" validate:"required" example:"123456"`
}

type ResetPassword struct {
	Email       string `json:"email" validate:"required,email" example:"user@example.com"`
	Code        string `json:"code" validate:"required" example:"123456"`
	NewPassword string `json:"new_password" validate:"required" example:"newStrongPassword123"`
}

