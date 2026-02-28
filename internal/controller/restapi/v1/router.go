// backend/internal/controller/restapi/v1/router.go
package v1

import (
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// NewTranslationRoutes -.
// func NewTranslationRoutes(apiV1Group fiber.Router, l logger.Interface) {
// 	r := &V1{l: l, v: validator.New(validator.WithRequiredStructEnabled())}

// 	translationGroup := apiV1Group.Group("/translation")

// 	{
// 		translationGroup.Get("/history", r.history)
// 		translationGroup.Post("/do-translate", r.doTranslate)
// 	}
// }

func NewCategoryRoutes(apiV1Group fiber.Router, c usecase.Category, l logger.Interface) {
	// Создаем экземпляр V1 с прокинутым юзкейсом категорий, логгером и валидатором
	r := &V1{c: c, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	categoryGroup := apiV1Group.Group("/categories")
	{
		categoryGroup.Get("", r.getCategories)
		categoryGroup.Get("/:id", r.getCategoryByID)

		// Админские роуты защищены middleware requireAdmin
		categoryGroup.Post("", r.requireAdmin, r.createCategory)
		categoryGroup.Patch("/:id", r.requireAdmin, r.updateCategory)
		categoryGroup.Delete("/:id", r.requireAdmin, r.deleteCategory)
	}
}
