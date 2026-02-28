// backend/internal/controller/restapi/v1/controller.go
package v1

import (
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/go-playground/validator/v10"
)

// V1 -.
type V1 struct {
	c usecase.Category // Добавили зависимость для Категорий
	l logger.Interface
	v *validator.Validate
}
