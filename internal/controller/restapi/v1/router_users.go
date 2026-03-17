package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

func NewUserRoutes(apiV1Group fiber.Router, u usecase.User, l logger.Interface, avatarBaseURL string) {
	r := &UsersV1{
		u:             u,
		l:             l,
		v:             validator.New(validator.WithRequiredStructEnabled()),
		avatarBaseURL: avatarBaseURL,
	}

	usersGroup := apiV1Group.Group("/users")
	{
		usersGroup.Post("", r.createUser)
		usersGroup.Post("/login", r.login)
		usersGroup.Post("/email-code/send", r.sendEmailVerificationCode)
		usersGroup.Post("/email-code/verify", r.verifyEmailVerificationCode)
		usersGroup.Delete("/:id", r.deleteUser)
		usersGroup.Get("/:id", r.getUser)
		usersGroup.Get("", r.listUsers)
		usersGroup.Put("/:id", r.updateUser)
		usersGroup.Post("/:id/avatar", r.uploadAvatar)
		usersGroup.Post("/password-reset/send-code", r.sendPasswordResetCode)
	}
}
