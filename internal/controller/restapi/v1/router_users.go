package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	authmw "github.com/sday-kenta/backend/internal/controller/restapi/middleware"
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
		usersGroup.Post("", authmw.RequireAdmin(), r.createUserByAdmin)
		usersGroup.Post("/email-code/send", r.sendEmailVerificationCode)
		usersGroup.Post("/email-code/verify", r.verifyEmailVerificationCode)
		usersGroup.Delete("/:id", authmw.RequireAdmin(), r.deleteUser)
		usersGroup.Get("/:id", authmw.RequireAuth(), r.getUser)
		usersGroup.Get("", authmw.RequireAdmin(), r.listUsers)
		usersGroup.Put("/:id", authmw.RequireAuth(), r.updateUser)
		usersGroup.Post("/:id/avatar", authmw.RequireAuth(), r.uploadAvatar)
		usersGroup.Post("/password-reset/send-code", r.sendPasswordResetCode)
		usersGroup.Post("/password-reset/reset", r.resetPasswordWithCode)
	}
}
