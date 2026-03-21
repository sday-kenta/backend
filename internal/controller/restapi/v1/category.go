// backend/internal/controller/restapi/v1/category.go

package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/categoryerr"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/entity"
)

func categoryErrorResponse(ctx *fiber.Ctx, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, categoryerr.ErrNotFound):
		return errorResponse(ctx, http.StatusNotFound, "category not found")
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}
}

// requireAdmin - middleware для проверки прав администратора
func (r *V1) requireAdmin(ctx *fiber.Ctx) error {
	role := ctx.Get("X-User-Role")
	if role != "admin" {
		return errorResponse(ctx, http.StatusForbidden, "access denied: admin role required")
	}
	return ctx.Next()
}

// @Summary     Get all categories
// @Description Returns a list of all active categories (rubrics)
// @ID          get-categories
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Success     200 {array} entity.Category "List of categories"
// @Failure     500 {object} response.Error
// @Router      /categories [get]
func (r *V1) getCategories(ctx *fiber.Ctx) error {
	categories, err := r.c.GetAll(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - getCategories")
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": categories})
}

// @Summary     Get category by ID
// @Description Returns a single active category by its ID
// @ID          get-category-by-id
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Success     200 {object} entity.Category "Single category"
// @Failure     400 {object} response.Error "Invalid ID format"
// @Failure     404 {object} response.Error "Category not found"
// @Failure     500 {object} response.Error
// @Router      /categories/{id} [get]
func (r *V1) getCategoryByID(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	category, err := r.c.GetByID(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - getCategoryByID")
		return categoryErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Create a new category
// @Description Creates a new category (Admin only)
// @ID          create-category
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Param       X-User-Role header string true "User role (must be 'admin')" default(admin)
// @Param       request body request.CreateCategory true "Data for new category"
// @Success     201 {object} entity.Category "Created category"
// @Failure     400 {object} response.Error "Invalid request body"
// @Failure     403 {object} response.Error "Access denied"
// @Failure     500 {object} response.Error
// @Router      /categories [post]
func (r *V1) createCategory(ctx *fiber.Ctx) error {
	var body request.CreateCategory

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	// Используем встроенный в V1 валидатор (как в translation.go)
	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - Validate")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request data: title is required")
	}

	category, err := r.c.Create(ctx.UserContext(), entity.CreateCategoryInput{
		Title:   body.Title,
		IconURL: body.IconURL,
	})
	if err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - Create")
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusCreated).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Update a category
// @Description Partially updates a category by ID (Admin only)
// @ID          update-category
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Param       X-User-Role header string true "User role (must be 'admin')" default(admin)
// @Param       request body request.UpdateCategory true "Data to update"
// @Success     200 {object} entity.Category "Updated category"
// @Failure     400 {object} response.Error "Invalid ID or request body"
// @Failure     403 {object} response.Error "Access denied"
// @Failure     404 {object} response.Error "Category not found"
// @Failure     500 {object} response.Error
// @Router      /categories/{id} [patch]
func (r *V1) updateCategory(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	var body request.UpdateCategory
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateCategory - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	category, err := r.c.Update(ctx.UserContext(), id, entity.UpdateCategoryInput{
		Title:   body.Title,
		IconURL: body.IconURL,
	})
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateCategory - Update")
		return categoryErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Delete a category
// @Description Soft-deletes a category by ID (Admin only)
// @ID          delete-category
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Param       X-User-Role header string true "User role (must be 'admin')" default(admin)
// @Success     200 {object} map[string]interface{} "Success message"
// @Failure     400 {object} response.Error "Invalid ID format"
// @Failure     403 {object} response.Error "Access denied"
// @Failure     404 {object} response.Error "Category not found"
// @Failure     500 {object} response.Error
// @Router      /categories/{id} [delete]
func (r *V1) deleteCategory(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	err = r.c.Delete(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategory - Delete")
		return categoryErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": "category deleted successfully"})
}
