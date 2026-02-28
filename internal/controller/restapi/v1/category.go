package v1

import (
	"net/http"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/gofiber/fiber/v2"
)

// requireAdmin - middleware для проверки прав администратора
func (r *V1) requireAdmin(ctx *fiber.Ctx) error {
	role := ctx.Get("X-User-Role")
	if role != "admin" {
		return errorResponse(ctx, http.StatusForbidden, "access denied: admin role required")
	}
	return ctx.Next()
}

// @Summary     Get categories
// @Description Get all active categories (rubrics)
// @ID          categories-get-all
// @Tags  	    categories
// @Accept      json
// @Produce     json
// @Success     200 {array} entity.Category
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

// getCategoryByID возвращает категорию по ID
func (r *V1) getCategoryByID(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	category, err := r.c.GetByID(ctx.UserContext(), id)
	if err != nil {
		return errorResponse(ctx, http.StatusNotFound, "category not found")
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// createCategoryReq структура запроса на создание
type createCategoryReq struct {
	Title   string `json:"title" validate:"required"` // Добавили тег валидации из шаблона
	IconURL string `json:"icon_url"`
}

// createCategory создает новую рубрику (только админ)
func (r *V1) createCategory(ctx *fiber.Ctx) error {
	var body createCategoryReq

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

// updateCategoryReq структура запроса на обновление
type updateCategoryReq struct {
	Title   *string `json:"title"`
	IconURL *string `json:"icon_url"`
}

// updateCategory обновляет поля рубрики (только админ)
func (r *V1) updateCategory(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	var body updateCategoryReq
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
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// deleteCategory выполняет soft-delete (только админ)
func (r *V1) deleteCategory(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	err = r.c.Delete(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategory - Delete")
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": "category deleted successfully"})
}
