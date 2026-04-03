package v1

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/pkg/logger"
	"github.com/sday-kenta/backend/pkg/objectstorage"
)

func userErrorResponse(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usererr.ErrNotFound):
		return errorResponse(ctx, http.StatusNotFound, "user not found")
	case errors.Is(err, usererr.ErrDuplicateLogin):
		return errorResponse(ctx, http.StatusConflict, "login already exists")
	case errors.Is(err, usererr.ErrDuplicateEmail):
		return errorResponse(ctx, http.StatusConflict, "email already exists")
	case errors.Is(err, usererr.ErrDuplicatePhone):
		return errorResponse(ctx, http.StatusConflict, "phone already exists")
	case errors.Is(err, usererr.ErrInvalidRole):
		return errorResponse(ctx, http.StatusBadRequest, "invalid role")
	case errors.Is(err, usererr.ErrInvalidPhone):
		return errorResponse(ctx, http.StatusBadRequest, err.Error())
	case errors.Is(err, usererr.ErrPasswordTooShort):
		return errorResponse(ctx, http.StatusBadRequest, err.Error())
	case errors.Is(err, usererr.ErrEmailNotVerified):
		return errorResponse(ctx, http.StatusForbidden, err.Error())
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "database error")
	}
}

// formatValidationError builds a human-readable validation error message.
func formatValidationError(err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok {
		msgs := make([]string, 0, len(errs))
		for _, fe := range errs {
			// Use JSON field name when possible, fall back to struct field.
			field := fe.Field()
			if fe.StructField() != "" && fe.Field() != "" {
				// Lowercase first letter to look closer to JSON name (simple heuristic).
				field = strings.ToLower(field[:1]) + field[1:]
			}

			msg := fmt.Sprintf("%s failed on '%s' rule", field, fe.Tag())
			if fe.Tag() == "oneof" {
				msg = fmt.Sprintf("%s must be one of [%s]", field, fe.Param())
			}
			msgs = append(msgs, msg)
		}

		return strings.Join(msgs, "; ")
	}

	return "invalid request body"
}

// UsersV1 handles user-related endpoints.
type UsersV1 struct {
	u             usecase.User
	l             logger.Interface
	v             *validator.Validate
	avatarBaseURL string
}

// @Summary     Create user
// @Description Create a new user
// @ID          create-user
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.CreateUser true "User data"
// @Success     201 {object} entity.User
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users [post]
func (r *UsersV1) createUser(ctx *fiber.Ctx) error {
	var body request.CreateUser

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createUser")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createUser")

		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	user, err := r.u.Create(
		ctx.UserContext(),
		entity.User{
			Login:      body.Login,
			Email:      body.Email,
			LastName:   body.LastName,
			FirstName:  body.FirstName,
			MiddleName: body.MiddleName,
			Phone:      body.Phone,
			City:       body.City,
			Street:     body.Street,
			House:      body.House,
			Apartment:  body.Apartment,
			IsBlocked:  body.IsBlocked,
			Role:       body.Role,
		},
		body.Password,
	)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createUser")
		return userErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusCreated).JSON(user)
}

// @Summary     Delete user
// @Description Delete user by ID
// @ID          delete-user
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       id path int true "User ID"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/{id} [delete]
func (r *UsersV1) deleteUser(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}

	if err = r.u.Delete(ctx.UserContext(), id); err != nil {
		r.l.Error(err, "restapi - v1 - deleteUser")
		return userErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Get user
// @Description Get user by ID
// @ID          get-user
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       id path int true "User ID"
// @Success     200 {object} entity.User
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/{id} [get]
func (r *UsersV1) getUser(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}

	user, err := r.u.GetByID(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - getUser")
		return userErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(user)
}

// @Summary     List users
// @Description Get all users
// @ID          list-users
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Success     200 {array} entity.User
// @Failure     500 {object} response.Error
// @Router      /users [get]
func (r *UsersV1) listUsers(ctx *fiber.Ctx) error {
	users, err := r.u.List(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - listUsers")

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusOK).JSON(users)
}

// @Summary     Update user
// @Description Update user fields (without password)
// @ID          update-user
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       id path int true "User ID"
// @Param       request body request.UpdateUser true "User data"
// @Success     200 {object} entity.User
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/{id} [put]
func (r *UsersV1) updateUser(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}

	var body request.UpdateUser

	if err = ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateUser")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err = r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateUser")

		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	user, err := r.u.Update(
		ctx.UserContext(),
		entity.User{
			ID:         id,
			Login:      body.Login,
			Email:      body.Email,
			LastName:   body.LastName,
			FirstName:  body.FirstName,
			MiddleName: body.MiddleName,
			Phone:      body.Phone,
			City:       body.City,
			Street:     body.Street,
			House:      body.House,
			Apartment:  body.Apartment,
			IsBlocked:  body.IsBlocked,
			Role:       body.Role,
		},
	)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateUser")
		return userErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(user)
}

// @Summary     Upload user avatar
// @Description Upload avatar image (JPEG/PNG) for a user
// @ID          upload-avatar
// @Tags  	    users
// @Accept      multipart/form-data
// @Produce     json
// @Param       id     path     int   true  "User ID"
// @Param       avatar formData file  true  "Avatar image (JPEG/PNG), max 2MB"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/{id}/avatar [post]
func (r *UsersV1) uploadAvatar(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}

	fileHeader, err := ctx.FormFile("avatar")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "avatar file is required")
	}

	const maxAvatarSize = 2 * 1024 * 1024 // 2MB
	if fileHeader.Size > maxAvatarSize {
		return errorResponse(ctx, http.StatusBadRequest, "avatar file too large (max 2MB)")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".png", ".jpg", ".jpeg":
	default:
		return errorResponse(ctx, http.StatusBadRequest, "avatar file must be PNG or JPG")
	}

	avatarKey := fmt.Sprintf("users/user-%d-%d%s", id, time.Now().UnixNano(), ext)

	storage, err := objectstorage.NewFromEnv(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar - NewFromEnv")
		return errorResponse(ctx, http.StatusInternalServerError, "avatar storage is not configured")
	}

	file, err := fileHeader.Open()
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar - FormFile.Open")
		return errorResponse(ctx, http.StatusInternalServerError, "failed to read avatar file")
	}
	defer func() {
		_ = file.Close()
	}()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err = storage.Upload(ctx.UserContext(), avatarKey, contentType, file); err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar - Upload")
		return errorResponse(ctx, http.StatusInternalServerError, "failed to upload avatar")
	}

	avatarValue := buildObjectURL(r.avatarBaseURL, avatarKey)

	if err = r.u.UpdateAvatar(ctx.UserContext(), id, avatarValue); err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar")
		_ = storage.Delete(ctx.UserContext(), avatarKey)
		return userErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Send password reset code
// @Description Send a password reset code to email
// @ID          send-password-reset-code
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.SendPasswordResetCode true "Email"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/password-reset/send-code [post]
func (r *UsersV1) sendPasswordResetCode(ctx *fiber.Ctx) error {
	var body request.SendPasswordResetCode

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - sendPasswordResetCode")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - sendPasswordResetCode")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	if err := r.u.SendPasswordResetCode(ctx.UserContext(), body.Email); err != nil {
		r.l.Error(err, "restapi - v1 - sendPasswordResetCode")
		return userErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Reset password with code
// @Description Verifies the code from email and sets a new password
// @ID          reset-password-with-code
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.ResetPasswordWithCode true "Email, code and new password"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/password-reset/reset [post]
func (r *UsersV1) resetPasswordWithCode(ctx *fiber.Ctx) error {
	var body request.ResetPasswordWithCode
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - resetPasswordWithCode")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}
	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - resetPasswordWithCode")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}
	if err := r.u.ResetPasswordWithCode(ctx.UserContext(), body.Email, body.Code, body.NewPassword); err != nil {
		r.l.Error(err, "restapi - v1 - resetPasswordWithCode")
		switch {
		case errors.Is(err, usererr.ErrInvalidCode), errors.Is(err, usererr.ErrCodeExpired):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		case errors.Is(err, usererr.ErrPasswordTooShort):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		case errors.Is(err, usererr.ErrNotFound):
			return errorResponse(ctx, http.StatusNotFound, "user not found")
		default:
			return userErrorResponse(ctx, err)
		}
	}
	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Send email verification code
// @Description Sends a verification code to email for registration or email change
// @ID          send-email-code
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.SendEmailVerificationCode true "Email and purpose"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/email-code/send [post]
func (r *UsersV1) sendEmailVerificationCode(ctx *fiber.Ctx) error {
	var body request.SendEmailVerificationCode

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - sendEmailVerificationCode")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - sendEmailVerificationCode")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	if err := r.u.SendEmailVerificationCode(ctx.UserContext(), body.Email, body.Purpose); err != nil {
		r.l.Error(err, "restapi - v1 - sendEmailVerificationCode")
		return userErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Verify email verification code
// @Description Verifies a code sent to email (one-time, with expiration)
// @ID          verify-email-code
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.VerifyEmailVerificationCode true "Email, purpose and code"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/email-code/verify [post]
func (r *UsersV1) verifyEmailVerificationCode(ctx *fiber.Ctx) error {
	var body request.VerifyEmailVerificationCode

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - verifyEmailVerificationCode")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - verifyEmailVerificationCode")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	if err := r.u.VerifyEmailVerificationCode(ctx.UserContext(), body.Email, body.Purpose, body.Code); err != nil {
		r.l.Error(err, "restapi - v1 - verifyEmailVerificationCode")
		switch {
		case errors.Is(err, usererr.ErrInvalidCode), errors.Is(err, usererr.ErrCodeExpired):
			return errorResponse(ctx, http.StatusBadRequest, err.Error())
		default:
			return userErrorResponse(ctx, err)
		}
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Login
// @Description Login by login/email/phone + password
// @ID          users-login
// @Tags  	    users
// @Accept      json
// @Produce     json
// @Param       request body request.Login true "Credentials"
// @Success     200 {object} entity.User
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /users/login [post]
func (r *UsersV1) login(ctx *fiber.Ctx) error {
	var body request.Login
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - usersLogin")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - usersLogin")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	user, err := r.u.Authenticate(ctx.UserContext(), body.Identifier, body.Password)
	if err != nil {
		r.l.Error(err, "restapi - v1 - usersLogin")
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
