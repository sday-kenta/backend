// backend/internal/controller/restapi/v1/category.go

package v1

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/categoryerr"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/pkg/objectstorage"
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

// @Summary     List categories
// @Description Returns all active categories. Category icon URLs are returned in the icon_url field when an icon is uploaded.
// @ID          list-categories
// @Tags        categories
// @Accept      json
// @Produce     json
// @Success     200 {object} map[string]interface{} "status + categories list"
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
// @Description Returns a single active category by its ID, including icon_url when an icon is uploaded.
// @ID          get-category-by-id
// @Tags        categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Success     200 {object} map[string]interface{} "status + category"
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

// @Summary     Create a category
// @Description Creates a new category. Category icons are uploaded separately via multipart/form-data endpoint. Admin only.
// @ID          create-category
// @Tags        categories
// @Accept      json
// @Produce     json
// @Param       request body request.CreateCategory true "Category payload"
// @Security    BearerAuth
// @Success     201 {object} map[string]interface{} "status + created category"
// @Failure     400 {object} response.Error "Invalid request body"
// @Failure     401 {object} response.Error "Authentication required"
// @Failure     403 {object} response.Error "Access denied"
// @Failure     500 {object} response.Error
// @Router      /categories [post]
func (r *V1) createCategory(ctx *fiber.Ctx) error {
	var body request.CreateCategory

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - Validate")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	category, err := r.c.Create(ctx.UserContext(), entity.CreateCategoryInput{Title: body.Title})
	if err != nil {
		r.l.Error(err, "restapi - v1 - createCategory - Create")
		return errorResponse(ctx, http.StatusInternalServerError, "database problems")
	}

	return ctx.Status(http.StatusCreated).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Update a category
// @Description Partially updates a category by ID. Category icons are managed via dedicated upload/delete endpoints. Admin only.
// @ID          update-category
// @Tags        categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Param       request body request.UpdateCategory true "Category fields to update"
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{} "status + updated category"
// @Failure     400 {object} response.Error "Invalid ID or request body"
// @Failure     401 {object} response.Error "Authentication required"
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

	category, err := r.c.Update(ctx.UserContext(), id, entity.UpdateCategoryInput{Title: body.Title})
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateCategory - Update")
		return categoryErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Upload category icon
// @Description Uploads a category icon to MinIO/S3 and stores its public URL in icon_url. Accepts PNG/JPG up to 2MB. Admin only.
// @ID          upload-category-icon
// @Tags        categories
// @Accept      multipart/form-data
// @Produce     json
// @Param       id path int true "Category ID"
// @Param       icon formData file true "Category icon (PNG/JPG, max 2MB)"
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{} "status + updated category"
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /categories/{id}/icon [post]
func (r *V1) uploadCategoryIcon(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	existing, err := r.c.GetByID(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadCategoryIcon - GetByID")
		return categoryErrorResponse(ctx, err)
	}

	fileHeader, err := ctx.FormFile("icon")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "icon file is required")
	}

	const maxIconSize = 2 * 1024 * 1024
	if fileHeader.Size > maxIconSize {
		return errorResponse(ctx, http.StatusBadRequest, "icon file too large (max 2MB)")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".png", ".jpg", ".jpeg":
	default:
		return errorResponse(ctx, http.StatusBadRequest, "icon file must be PNG or JPG")
	}

	storage, err := objectstorage.NewFromEnv(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadCategoryIcon - NewFromEnv")
		return errorResponse(ctx, http.StatusInternalServerError, "icon storage is not configured")
	}

	file, err := fileHeader.Open()
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadCategoryIcon - FormFile.Open")
		return errorResponse(ctx, http.StatusInternalServerError, "failed to read icon file")
	}
	defer func() { _ = file.Close() }()

	iconKey := fmt.Sprintf("categories/%d/%d%s", id, time.Now().UnixNano(), ext)
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err = storage.Upload(ctx.UserContext(), iconKey, contentType, file); err != nil {
		r.l.Error(err, "restapi - v1 - uploadCategoryIcon - Upload")
		return errorResponse(ctx, http.StatusInternalServerError, "failed to upload icon")
	}

	category, err := r.c.UpdateIcon(ctx.UserContext(), id, buildObjectURL(r.categoryMediaBaseURL, iconKey))
	if err != nil {
		_ = storage.Delete(ctx.UserContext(), iconKey)
		r.l.Error(err, "restapi - v1 - uploadCategoryIcon - UpdateIcon")
		return categoryErrorResponse(ctx, err)
	}

	oldIconKey := objectKeyFromStoredURL(r.categoryMediaBaseURL, existing.IconURL)
	if oldIconKey != "" && oldIconKey != iconKey {
		if err = storage.Delete(ctx.UserContext(), oldIconKey); err != nil {
			r.l.Error(err, "restapi - v1 - uploadCategoryIcon - DeleteOldIcon")
		}
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": category})
}

// @Summary     Delete category icon
// @Description Deletes a category icon from MinIO/S3 and clears icon_url. Admin only.
// @ID          delete-category-icon
// @Tags        categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Security    BearerAuth
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /categories/{id}/icon [delete]
func (r *V1) deleteCategoryIcon(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	existing, err := r.c.GetByID(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategoryIcon - GetByID")
		return categoryErrorResponse(ctx, err)
	}

	if _, err = r.c.DeleteIcon(ctx.UserContext(), id); err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategoryIcon - DeleteIcon")
		return categoryErrorResponse(ctx, err)
	}

	oldIconKey := objectKeyFromStoredURL(r.categoryMediaBaseURL, existing.IconURL)
	if oldIconKey != "" {
		storage, storageErr := objectstorage.NewFromEnv(ctx.UserContext())
		if storageErr == nil {
			if err = storage.Delete(ctx.UserContext(), oldIconKey); err != nil {
				r.l.Error(err, "restapi - v1 - deleteCategoryIcon - DeleteObject")
			}
		}
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Delete a category
// @Description Soft-deletes a category by ID. Admin only.
// @ID          delete-category
// @Tags        categories
// @Accept      json
// @Produce     json
// @Param       id path int true "Category ID"
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{} "Success message"
// @Failure     400 {object} response.Error "Invalid ID format"
// @Failure     401 {object} response.Error "Authentication required"
// @Failure     403 {object} response.Error "Access denied"
// @Failure     404 {object} response.Error "Category not found"
// @Failure     500 {object} response.Error
// @Router      /categories/{id} [delete]
func (r *V1) deleteCategory(ctx *fiber.Ctx) error {
	id, err := ctx.ParamsInt("id")
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id format")
	}

	existing, err := r.c.GetByID(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategory - GetByID")
		return categoryErrorResponse(ctx, err)
	}

	err = r.c.Delete(ctx.UserContext(), id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteCategory - Delete")
		return categoryErrorResponse(ctx, err)
	}

	oldIconKey := objectKeyFromStoredURL(r.categoryMediaBaseURL, existing.IconURL)
	if oldIconKey != "" {
		storage, storageErr := objectstorage.NewFromEnv(ctx.UserContext())
		if storageErr == nil {
			if err = storage.Delete(ctx.UserContext(), oldIconKey); err != nil {
				r.l.Error(err, "restapi - v1 - deleteCategory - DeleteObject")
			}
		}
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": "category deleted successfully"})
}
