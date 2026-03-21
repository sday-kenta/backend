// backend/internal/controller/restapi/v1/request/category.go

package request

// CreateCategory описывает HTTP-запрос на создание рубрики.
type CreateCategory struct {
	Title string `json:"title" validate:"required"`
}

// UpdateCategory описывает HTTP-запрос на обновление рубрики (PATCH).
type UpdateCategory struct {
	Title *string `json:"title"`
}
