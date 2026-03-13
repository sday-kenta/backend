package restapi

import (
	"net/http"

	"github.com/evrone/go-clean-template/config"
	_ "github.com/evrone/go-clean-template/docs"
	"github.com/evrone/go-clean-template/internal/controller/restapi/middleware"
	v1 "github.com/evrone/go-clean-template/internal/controller/restapi/v1"
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// NewRouter configures HTTP routes.
func NewRouter(app *fiber.App, cfg *config.Config, c usecase.Category, g usecase.Geo, l logger.Interface) {
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))

	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	app.Get("/healthz", func(ctx *fiber.Ctx) error { return ctx.SendStatus(http.StatusOK) })

	apiV1Group := app.Group("/v1")
	{
		v1.NewCategoryRoutes(apiV1Group, c, l)
		v1.NewGeoRoutes(apiV1Group, g, l)
	}
}
