package restapi

import (
	"net/http"
	"strings"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/sday-kenta/backend/config"
	docs "github.com/sday-kenta/backend/docs"
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
func NewRouter(app *fiber.App, cfg *config.Config, c usecase.Category, g usecase.Geo, u usecase.User, i usecase.Incident, l logger.Interface) {
	app.Use(middleware.Logger(l))
	app.Use(middleware.Recovery(l))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-User-Role, X-User-ID",
	}))

	if cfg.Metrics.Enabled {
		prometheus := fiberprometheus.New("backend")
		prometheus.RegisterAt(app, "/metrics")
		app.Use(prometheus.Middleware)
	}

	if cfg.Swagger.Enabled {
		applySwaggerConfig(cfg)
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	app.Get("/healthz", func(ctx *fiber.Ctx) error { return ctx.SendStatus(http.StatusOK) })

	apiV1Group := app.Group("/v1")
	{
		categoryMediaBaseURL := cfg.CDN.CategoryMediaBaseURL
		if categoryMediaBaseURL == "" {
			categoryMediaBaseURL = cfg.CDN.IncidentMediaBaseURL
		}
		if categoryMediaBaseURL == "" {
			categoryMediaBaseURL = cfg.CDN.AvatarBaseURL
		}
		v1.NewCategoryRoutes(apiV1Group, c, l, categoryMediaBaseURL)
		v1.NewGeoRoutes(apiV1Group, g, l)
		v1.NewUserRoutes(apiV1Group, u, l, cfg.CDN.AvatarBaseURL)
		incidentMediaBaseURL := cfg.CDN.IncidentMediaBaseURL
		if incidentMediaBaseURL == "" {
			incidentMediaBaseURL = cfg.CDN.AvatarBaseURL
		}
		v1.NewIncidentRoutes(apiV1Group, i, l, incidentMediaBaseURL)
		v1.NewAuthRoutes(apiV1Group, u, l)
	}
}

func applySwaggerConfig(cfg *config.Config) {
	if host := strings.TrimSpace(cfg.Swagger.Host); host != "" {
		docs.SwaggerInfo.Host = host
	}

	if basePath := strings.TrimSpace(cfg.Swagger.BasePath); basePath != "" {
		docs.SwaggerInfo.BasePath = basePath
	}

	if schemes := parseSwaggerSchemes(cfg.Swagger.Schemes); len(schemes) > 0 {
		docs.SwaggerInfo.Schemes = schemes
	}
}

func parseSwaggerSchemes(raw string) []string {
	parts := strings.Split(raw, ",")
	schemes := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		schemes = append(schemes, part)
	}

	return schemes
}
