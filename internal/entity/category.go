package entity

// Category представляет рубрику инцидента (Парковки, Просрочка).
type Category struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	IconURL  string `json:"icon_url,omitempty"`
	IsActive bool   `json:"-"` // Скрыто из JSON ответа
}
