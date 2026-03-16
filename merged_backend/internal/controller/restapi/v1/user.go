package v1

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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

	// Limit size to 2MB to avoid storing huge files in DB.
	const maxAvatarSize = 2 * 1024 * 1024 // 2MB
	if fileHeader.Size > maxAvatarSize {
		return errorResponse(ctx, http.StatusBadRequest, "avatar file too large (max 2MB)")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".png", ".jpg", ".jpeg":
		// ok
	default:
		return errorResponse(ctx, http.StatusBadRequest, "avatar file must be PNG or JPG")
	}

	avatarKey := fmt.Sprintf("user-%d-%d%s", id, time.Now().UnixNano(), ext)

	bucket := os.Getenv("AWS_S3_BUCKET")
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if bucket == "" || endpoint == "" || region == "" || accessKey == "" || secretKey == "" {
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

	awsCfg, err := awscfg.LoadDefaultConfig(
		ctx.UserContext(),
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		),
		awscfg.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, _ string, _ ...interface{}) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{
							URL:               endpoint,
							HostnameImmutable: true,
						}, nil
					}

					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar - LoadDefaultConfig")

		return errorResponse(ctx, http.StatusInternalServerError, "failed to initialize avatar storage")
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s3Client.PutObject(ctx.UserContext(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(avatarKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar - PutObject")

		return errorResponse(ctx, http.StatusInternalServerError, "failed to upload avatar")
	}

	avatarValue := avatarKey
	if r.avatarBaseURL != "" {
		base := strings.TrimRight(r.avatarBaseURL, "/")
		avatarValue = fmt.Sprintf("%s/%s", base, avatarKey)
	}

	if err = r.u.UpdateAvatar(ctx.UserContext(), id, avatarValue); err != nil {
		r.l.Error(err, "restapi - v1 - uploadAvatar")
		return userErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}


