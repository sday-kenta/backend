package restapi

import (
	"net/http"

	"github.com/sday-kenta/backend/config"
	_ "github.com/sday-kenta/backend/docs"
	"github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	v1 "github.com/sday-kenta/backend/internal/controller/restapi/v1"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
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
