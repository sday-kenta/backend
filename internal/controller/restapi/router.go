package restapi

import (
	"net/http"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/sday-kenta/backend/config"
	_ "github.com/sday-kenta/backend/docs"
	"github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	v1 "github.com/sday-kenta/backend/internal/controller/restapi/v1"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

// NewRouter configures HTTP routes.
// CORS is kept globally because browser-based Swagger UI and any separate frontend origin
// will otherwise be blocked by the browser before reaching the API.
// Metrics are mounted once for the whole service so both users and maps endpoints are observed.
// Swagger spec:
// @title       Backend API
// @description Combined users and maps service API
// @version     1.0
// @host        localhost:8080
// @BasePath    /v1
func NewRouter(app *fiber.App, cfg *config.Config, c usecase.Category, g usecase.Geo, u usecase.User, l logger.Interface) {
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-User-Role",
	}))

	if cfg.Metrics.Enabled {
		prometheus := fiberprometheus.New("backend")
		prometheus.RegisterAt(app, "/metrics")
		app.Use(prometheus.Middleware)
	}

	if cfg.Swagger.Enabled {
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	app.Get("/healthz", func(ctx *fiber.Ctx) error { return ctx.SendStatus(http.StatusOK) })

	apiV1Group := app.Group("/v1")
	{
		v1.NewCategoryRoutes(apiV1Group, c, l)
		v1.NewGeoRoutes(apiV1Group, g, l)
		v1.NewUserRoutes(apiV1Group, u, l, cfg.CDN.AvatarBaseURL)
		v1.NewAuthRoutes(apiV1Group, u, l)
	}
}
