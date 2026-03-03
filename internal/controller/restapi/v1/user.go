package v1

import (
	"net/http"
	"strconv"

	"github.com/evrone/go-clean-template/internal/controller/restapi/v1/request"
	"github.com/evrone/go-clean-template/internal/controller/restapi/v1/response"
	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/gofiber/fiber/v2"
)

// UsersV1 handles user-related endpoints.
type UsersV1 struct {
	u usecase.User
	l logger.Interface
	v *validator.Validate
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

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
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
			IsAdmin:    body.IsAdmin,
		},
		body.Password,
	)
	if err != nil {
		r.l.Error(err, "restapi - v1 - createUser")

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
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

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
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

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
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

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
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
			IsAdmin:    body.IsAdmin,
		},
	)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateUser")

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusOK).JSON(user)
}

