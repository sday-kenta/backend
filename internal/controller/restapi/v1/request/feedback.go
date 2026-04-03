package request

type SendFeedback struct {
	Message string `json:"message" validate:"required,min=10,max=8000" example:"Текст обращения"`
	Email   string `json:"email,omitempty" validate:"omitempty,email" example:"user@example.com"`
	Name    string `json:"name,omitempty" validate:"max=200" example:"Иван"`
}
