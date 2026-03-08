package v1

import (
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func NewCategoryRoutes(apiV1Group fiber.Router, c usecase.Category, l logger.Interface) {
	r := &V1{c: c, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	categoryGroup := apiV1Group.Group("/categories")
	{
		categoryGroup.Get("", r.getCategories)
		categoryGroup.Get("/:id", r.getCategoryByID)
		categoryGroup.Post("", r.requireAdmin, r.createCategory)
		categoryGroup.Patch("/:id", r.requireAdmin, r.updateCategory)
		categoryGroup.Delete("/:id", r.requireAdmin, r.deleteCategory)
	}
}

func NewGeoRoutes(apiV1Group fiber.Router, g usecase.Geo, l logger.Interface) {
	r := &V1{g: g, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	mapsGroup := apiV1Group.Group("/maps")
	{
		mapsGroup.Get("/reverse", r.reverseGeocode)
		mapsGroup.Get("/search", r.searchAddresses)
	}
}
