package request

type SendEmailVerificationCode struct {
	Email   string `json:"email" validate:"required,email" example:"user@example.com"`
	Purpose string `json:"purpose" validate:"required,oneof=register change_email" example:"register"`
}

type VerifyEmailVerificationCode struct {
	Email   string `json:"email" validate:"required,email" example:"user@example.com"`
	Purpose string `json:"purpose" validate:"required,oneof=register change_email" example:"register"`
	Code    string `json:"code" validate:"required" example:"123456"`
}

