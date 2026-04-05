package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	authmw "github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

func NewCategoryRoutes(apiV1Group fiber.Router, c usecase.Category, l logger.Interface, categoryMediaBaseURL string) {
	r := &V1{
		c:                    c,
		l:                    l,
		v:                    validator.New(validator.WithRequiredStructEnabled()),
		categoryMediaBaseURL: categoryMediaBaseURL,
	}

	categoryGroup := apiV1Group.Group("/categories")
	{
		categoryGroup.Get("", r.getCategories)
		categoryGroup.Get("/:id", r.getCategoryByID)
		categoryGroup.Post("", authmw.RequireAdmin(), r.createCategory)
		categoryGroup.Patch("/:id", authmw.RequireAdmin(), r.updateCategory)
		categoryGroup.Post("/:id/icon", authmw.RequireAdmin(), r.uploadCategoryIcon)
		categoryGroup.Delete("/:id/icon", authmw.RequireAdmin(), r.deleteCategoryIcon)
		categoryGroup.Delete("/:id", authmw.RequireAdmin(), r.deleteCategory)
	}
}

func NewGeoRoutes(apiV1Group fiber.Router, g usecase.Geo, l logger.Interface) {
	r := &V1{g: g, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	mapsGroup := apiV1Group.Group("/maps")
	{
		mapsGroup.Get("/reverse", r.reverseGeocode)
		mapsGroup.Get("/search", r.searchAddresses)
		mapsGroup.Post("/reload-cities", authmw.RequireAdmin(), r.reloadCities)
	}
}
