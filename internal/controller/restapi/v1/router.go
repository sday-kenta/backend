package v1

import (
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// NewTranslationRoutes -.
func NewTranslationRoutes(apiV1Group fiber.Router, t usecase.Translation, l logger.Interface) {
	r := &V1{t: t, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	translationGroup := apiV1Group.Group("/translation")

	{
		translationGroup.Get("/history", r.history)
		translationGroup.Post("/do-translate", r.doTranslate)
	}
}

// NewUserRoutes registers user routes.
func NewUserRoutes(apiV1Group fiber.Router, u usecase.User, l logger.Interface, avatarBaseURL string) {
	r := &UsersV1{
		u:             u,
		l:             l,
		v:             validator.New(validator.WithRequiredStructEnabled()),
		avatarBaseURL: avatarBaseURL,
	}

	usersGroup := apiV1Group.Group("/users")

	{
		usersGroup.Post("/", r.createUser)
		usersGroup.Delete("/:id", r.deleteUser)
		usersGroup.Get("/:id", r.getUser)
		usersGroup.Get("/", r.listUsers)
		usersGroup.Put("/:id", r.updateUser)
		usersGroup.Post("/:id/avatar", r.uploadAvatar)
	}
}
