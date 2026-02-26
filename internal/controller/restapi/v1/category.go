package v1

import (
	"net/http"

	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

type categoryRoutes struct {
	c usecase.Category
	l logger.Interface
}

func NewCategoryRoutes(handler fiber.Router, c usecase.Category, l logger.Interface) {
	r := &categoryRoutes{c, l}

	h := handler.Group("/categories")
	{
		h.Get("", r.getAll)
	}
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
func (r *categoryRoutes) getAll(ctx *fiber.Ctx) error {
	categories, err := r.c.GetAll(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - category - getAll")

		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	// Возвращаем JSON в формате, удобном для фронтенда
	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   categories,
	})
}
