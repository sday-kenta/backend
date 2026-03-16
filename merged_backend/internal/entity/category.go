package entity

// Category представляет рубрику инцидента.
type Category struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	IconURL  string `json:"icon_url,omitempty"`
	IsActive bool   `json:"-"`
}

// CreateCategoryInput данные для создания.
type CreateCategoryInput struct {
	Title   string
	IconURL string
}

// UpdateCategoryInput данные для обновления (PATCH).
// Используем указатели, чтобы понимать, было ли поле передано в JSON.
type UpdateCategoryInput struct {
	Title   *string
	IconURL *string
}
