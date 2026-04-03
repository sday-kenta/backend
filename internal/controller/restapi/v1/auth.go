package v1

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

type AuthV1 struct {
	u usecase.User
	l logger.Interface
	v *validator.Validate
}

func NewAuthRoutes(apiV1Group fiber.Router, u usecase.User, l logger.Interface) {
	r := &AuthV1{
		u: u,
		l: l,
		v: validator.New(validator.WithRequiredStructEnabled()),
	}

	authGroup := apiV1Group.Group("/auth")
	{
		authGroup.Post("/login", r.login)
	}
}

// @Summary     Login
// @Description Login by login/email/phone + password
// @ID          login
// @Tags  	    auth
// @Accept      json
// @Produce     json
// @Param       request body request.Login true "Credentials"
// @Success     200 {object} entity.User
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /auth/login [post]
func (r *AuthV1) login(ctx *fiber.Ctx) error {
	var body request.Login
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - login")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - login")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	user, err := r.u.Authenticate(ctx.UserContext(), body.Identifier, body.Password)
	if err != nil {
		r.l.Error(err, "restapi - v1 - login")
		switch {
		case errors.Is(err, usererr.ErrInvalidCredentials):
			return errorResponse(ctx, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, usererr.ErrUserBlocked):
			return errorResponse(ctx, http.StatusForbidden, "user is blocked")
		case errors.Is(err, usererr.ErrEmailNotVerified):
			return errorResponse(ctx, http.StatusForbidden, err.Error())
		default:
			return userErrorResponse(ctx, err)
		}
	}

	return ctx.Status(http.StatusOK).JSON(user)
}

