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
	authmw "github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/response"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/incidenterr"
	"github.com/sday-kenta/backend/internal/usecase"
	pushuc "github.com/sday-kenta/backend/internal/usecase/push"
	"github.com/sday-kenta/backend/pkg/logger"
	"github.com/sday-kenta/backend/pkg/objectstorage"
)

// IncidentsV1 handles incidents/messages API.
type IncidentsV1 struct {
	i            usecase.Incident
	p            usecase.Push
	l            logger.Interface
	v            *validator.Validate
	mediaBaseURL string
}

type requester struct {
	UserID  int64
	HasUser bool
	IsAdmin bool
}

func requesterFromCtx(ctx *fiber.Ctx) (requester, error) {
	if user, ok := authmw.CurrentUser(ctx); ok {
		return requester{
			UserID:  user.UserID,
			HasUser: true,
			IsAdmin: user.IsAdmin(),
		}, nil
	}

	return requester{}, nil
}

func requireRequester(ctx *fiber.Ctx) (requester, error) {
	r, err := requesterFromCtx(ctx)
	if err != nil {
		return requester{}, err
	}
	if !r.HasUser {
		return requester{}, fmt.Errorf("authentication required")
	}

	return r, nil
}

func incidentErrorResponse(ctx *fiber.Ctx, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, incidenterr.ErrNotFound):
		return errorResponse(ctx, http.StatusNotFound, "incident not found")
	case errors.Is(err, incidenterr.ErrPhotoNotFound):
		return errorResponse(ctx, http.StatusNotFound, "incident photo not found")
	case errors.Is(err, incidenterr.ErrForbidden):
		return errorResponse(ctx, http.StatusForbidden, "access denied")
	case errors.Is(err, incidenterr.ErrInvalidStatus):
		return errorResponse(ctx, http.StatusBadRequest, "invalid incident status")
	case errors.Is(err, incidenterr.ErrInvalidCoordinates):
		return errorResponse(ctx, http.StatusBadRequest, "both latitude and longitude must be provided")
	case errors.Is(err, incidenterr.ErrLocationRequired):
		return errorResponse(ctx, http.StatusBadRequest, "address or coordinates are required")
	case errors.Is(err, incidenterr.ErrCategoryNotFound):
		return errorResponse(ctx, http.StatusBadRequest, "category not found")
	case errors.Is(err, incidenterr.ErrDocumentEmailEmpty):
		return errorResponse(ctx, http.StatusBadRequest, "email is required")
	case errors.Is(err, incidenterr.ErrRequesterNotFound):
		return errorResponse(ctx, http.StatusForbidden, "requester not found")
	case errors.Is(err, entity.ErrOutOfAllowedZone):
		return errorResponse(ctx, http.StatusUnprocessableEntity, err.Error())
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}
}

// @Summary     Создать инцидент
// @Description Создает новое сообщение об инциденте для авторизованного пользователя. Если статус не указан, инцидент сохраняется в review. Если обычный пользователь передает published, статус понижается до review. Администратор может сохранить published сразу.
// @ID          create-incident
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       request body request.CreateIncident true "Данные инцидента"
// @Security    BearerAuth
// @Success     201 {object} response.Incident
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents [post]
func (r *IncidentsV1) createIncident(ctx *fiber.Ctx) error {
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	var body request.CreateIncident
	if err = ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - createIncident - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}
	if err = r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - createIncident - Validate")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	incident, err := r.i.Create(ctx.UserContext(), requester.UserID, requester.IsAdmin, entity.CreateIncidentInput{
		CategoryID:     body.CategoryID,
		Title:          body.Title,
		Description:    body.Description,
		Status:         body.Status,
		DepartmentName: body.DepartmentName,
		City:           body.City,
		Street:         body.Street,
		House:          body.House,
		AddressText:    body.AddressText,
		Latitude:       body.Latitude,
		Longitude:      body.Longitude,
	})
	if err != nil {
		r.l.Error(err, "restapi - v1 - createIncident")
		return incidentErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusCreated).JSON(toIncidentResponse(incident))
}

// @Summary     Получить список инцидентов
// @Description Возвращает общий список инцидентов. Для обычного пользователя и анонимного запроса возвращаются только published. Для администратора по умолчанию возвращаются published и review, а query-параметр status можно передавать несколько раз, чтобы выбрать нужные статусы.
// @ID          list-incidents
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       Authorization header string false "Bearer access token. Для admin включает расширенный список и фильтры по статусам."
// @Param       status query []string false "Фильтр по статусам для admin; параметр можно повторять" collectionFormat(multi)
// @Param       category_id query int false "ID категории"
// @Success     200 {array} response.Incident
// @Failure     400 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents [get]
func (r *IncidentsV1) listIncidents(ctx *fiber.Ctx) error {
	requester, err := requesterFromCtx(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	var categoryID *int
	if raw := strings.TrimSpace(ctx.Query("category_id")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return errorResponse(ctx, http.StatusBadRequest, "invalid category_id")
		}
		categoryID = &parsed
	}

	filter := entity.IncidentFilter{CategoryID: categoryID, OnlyPublished: true}
	if requester.IsAdmin {
		statuses, parseErr := parseIncidentStatusFilters(ctx.Context().QueryArgs().PeekMulti("status"))
		if parseErr != nil {
			return errorResponse(ctx, http.StatusBadRequest, "invalid status")
		}
		filter.OnlyPublished = false
		if len(statuses) == 0 {
			filter.Statuses = []string{entity.IncidentStatusPublished, entity.IncidentStatusReview}
		} else {
			filter.Statuses = statuses
		}
	}

	incidents, err := r.i.List(ctx.UserContext(), filter)
	if err != nil {
		r.l.Error(err, "restapi - v1 - listIncidents")
		return incidentErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(toIncidentResponses(incidents))
}

func parseIncidentStatusFilters(values [][]byte) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	statuses := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, rawValue := range values {
		status := strings.TrimSpace(strings.ToLower(string(rawValue)))
		if status == "" {
			continue
		}
		switch status {
		case entity.IncidentStatusDraft, entity.IncidentStatusReview, entity.IncidentStatusPublished:
		default:
			return nil, incidenterr.ErrInvalidStatus
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// @Summary     Получить мои инциденты
// @Description Возвращает список инцидентов текущего пользователя, включая draft, review и published.
// @ID          list-my-incidents
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       status query string false "Фильтр по статусу" Enums(draft,review,published,all)
// @Param       category_id query int false "ID категории"
// @Security    BearerAuth
// @Success     200 {array} response.Incident
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /my/incidents [get]
func (r *IncidentsV1) listMyIncidents(ctx *fiber.Ctx) error {
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	var (
		status     *string
		categoryID *int
	)
	if rawStatus := strings.TrimSpace(strings.ToLower(ctx.Query("status"))); rawStatus != "" {
		switch rawStatus {
		case "all":
		case entity.IncidentStatusDraft, entity.IncidentStatusReview, entity.IncidentStatusPublished:
			status = &rawStatus
		default:
			return errorResponse(ctx, http.StatusBadRequest, "invalid status")
		}
	}
	if raw := strings.TrimSpace(ctx.Query("category_id")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return errorResponse(ctx, http.StatusBadRequest, "invalid category_id")
		}
		categoryID = &parsed
	}

	incidents, err := r.i.List(ctx.UserContext(), entity.IncidentFilter{UserID: &requester.UserID, Status: status, CategoryID: categoryID})
	if err != nil {
		r.l.Error(err, "restapi - v1 - listMyIncidents")
		return incidentErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(toIncidentResponses(incidents))
}

// @Summary     Получить инцидент по ID
// @Description Возвращает детальную карточку инцидента. Published-инциденты доступны всем. Draft и review доступны только автору или администратору.
// @ID          get-incident
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Param       Authorization header string false "Bearer access token. Нужен для доступа к draft/review инцидентам."
// @Success     200 {object} response.Incident
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id} [get]
func (r *IncidentsV1) getIncident(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}

	requester, err := requesterFromCtx(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	incident, err := r.i.GetByID(ctx.UserContext(), requester.UserID, requester.IsAdmin, id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - getIncident")
		return incidentErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(toIncidentResponse(incident))
}

// @Summary     Обновить инцидент
// @Description Обновляет сообщение об инциденте. Доступно только автору или администратору. Если обычный пользователь передает published, статус понижается до review. Администратор может сохранить published сразу, но не может редактировать чужие draft-инциденты.
// @ID          update-incident
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Param       request body request.UpdateIncident true "Поля для обновления инцидента"
// @Security    BearerAuth
// @Success     200 {object} response.Incident
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id} [patch]
func (r *IncidentsV1) updateIncident(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}
	before, err := r.i.GetByID(ctx.UserContext(), requester.UserID, requester.IsAdmin, id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateIncident - GetByID")
		return incidentErrorResponse(ctx, err)
	}

	var body request.UpdateIncident
	if err = ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateIncident - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}
	if err = r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateIncident - Validate")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	incident, err := r.i.Update(ctx.UserContext(), requester.UserID, requester.IsAdmin, id, entity.UpdateIncidentInput{
		CategoryID:     body.CategoryID,
		Title:          body.Title,
		Description:    body.Description,
		Status:         body.Status,
		DepartmentName: body.DepartmentName,
		City:           body.City,
		Street:         body.Street,
		House:          body.House,
		AddressText:    body.AddressText,
		Latitude:       body.Latitude,
		Longitude:      body.Longitude,
	})
	if err != nil {
		r.l.Error(err, "restapi - v1 - updateIncident")
		return incidentErrorResponse(ctx, err)
	}
	r.notifyIncidentStatusChanged(ctx, before, incident, requester.UserID, "updateIncident")

	return ctx.Status(http.StatusOK).JSON(toIncidentResponse(incident))
}

func (r *IncidentsV1) notifyIncidentStatusChanged(ctx *fiber.Ctx, before, after entity.Incident, actorUserID int64, action string) {
	if notification, ok := pushuc.BuildIncidentStatusNotification(before, after, actorUserID); ok {
		if notifyErr := r.p.NotifyIncidentStatusChanged(ctx.UserContext(), notification); notifyErr != nil {
			r.l.Error(fmt.Errorf("restapi - v1 - %s - NotifyIncidentStatusChanged: %w", action, notifyErr))
		}
	}
}

// @Summary     Удалить инцидент
// @Description Удаляет инцидент и связанные с ним фотографии. Доступно только автору или администратору.
// @ID          delete-incident
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Security    BearerAuth
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id} [delete]
func (r *IncidentsV1) deleteIncident(ctx *fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	photos, err := r.i.Delete(ctx.UserContext(), requester.UserID, requester.IsAdmin, id)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteIncident")
		return incidentErrorResponse(ctx, err)
	}

	storage, storageErr := objectstorage.NewFromEnv(ctx.UserContext())
	if storageErr == nil {
		for _, photo := range photos {
			if photo.FileKey == "" {
				continue
			}
			if err = storage.Delete(ctx.UserContext(), photo.FileKey); err != nil {
				r.l.Error(err, "restapi - v1 - deleteIncident - DeleteObject")
			}
		}
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Загрузить фотографии инцидента
// @Description Загружает одну или несколько фотографий инцидента. Используй multipart/form-data с повторяемым полем photos. Доступно только автору инцидента. Администратор не может загружать фотографии в чужие инциденты.
// @ID          upload-incident-photos
// @Tags        incidents
// @Accept      multipart/form-data
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Param       photos formData file true "Фотографии инцидента в формате JPEG/PNG; поле можно передавать несколько раз"
// @Security    BearerAuth
// @Success     201 {array} response.IncidentPhoto
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id}/photos [post]
func (r *IncidentsV1) uploadIncidentPhotos(ctx *fiber.Ctx) error {
	incidentID, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}
	if _, err = r.i.GetByID(ctx.UserContext(), requester.UserID, requester.IsAdmin, incidentID); err != nil {
		r.l.Error(err, "restapi - v1 - uploadIncidentPhotos - GetByID")
		return incidentErrorResponse(ctx, err)
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "photos are required")
	}
	files := form.File["photos"]
	if len(files) == 0 {
		return errorResponse(ctx, http.StatusBadRequest, "photos are required")
	}

	storage, err := objectstorage.NewFromEnv(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - uploadIncidentPhotos - NewFromEnv")
		return errorResponse(ctx, http.StatusInternalServerError, "photo storage is not configured")
	}

	photos := make([]response.IncidentPhoto, 0, len(files))
	for _, fileHeader := range files {
		const maxPhotoSize = 5 * 1024 * 1024 // 5MB
		if fileHeader.Size > maxPhotoSize {
			return errorResponse(ctx, http.StatusBadRequest, "incident photo too large (max 5MB)")
		}

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".png", ".jpg", ".jpeg":
		default:
			return errorResponse(ctx, http.StatusBadRequest, "incident photo must be PNG or JPG")
		}

		file, openErr := fileHeader.Open()
		if openErr != nil {
			r.l.Error(openErr, "restapi - v1 - uploadIncidentPhotos - FormFile.Open")
			return errorResponse(ctx, http.StatusInternalServerError, "failed to read incident photo")
		}

		photoKey := fmt.Sprintf("incidents/%d/%d%s", incidentID, time.Now().UnixNano(), ext)
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err = storage.Upload(ctx.UserContext(), photoKey, contentType, file); err != nil {
			_ = file.Close()
			r.l.Error(err, "restapi - v1 - uploadIncidentPhotos - Upload")
			return errorResponse(ctx, http.StatusInternalServerError, "failed to upload incident photo")
		}
		_ = file.Close()

		photoURL := buildObjectURL(r.mediaBaseURL, photoKey)

		photo, createErr := r.i.CreatePhoto(ctx.UserContext(), requester.UserID, requester.IsAdmin, incidentID, entity.IncidentPhoto{
			FileKey:     photoKey,
			FileURL:     photoURL,
			ContentType: contentType,
			SizeBytes:   fileHeader.Size,
		})
		if createErr != nil {
			_ = storage.Delete(ctx.UserContext(), photoKey)
			r.l.Error(createErr, "restapi - v1 - uploadIncidentPhotos - CreatePhoto")
			return incidentErrorResponse(ctx, createErr)
		}

		photos = append(photos, toIncidentPhotoResponse(photo))
	}

	return ctx.Status(http.StatusCreated).JSON(photos)
}

// @Summary     Удалить фотографию инцидента
// @Description Удаляет фотографию инцидента. Доступно только автору или администратору.
// @ID          delete-incident-photo
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Param       photoId path int true "ID фотографии"
// @Security    BearerAuth
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id}/photos/{photoId} [delete]
func (r *IncidentsV1) deleteIncidentPhoto(ctx *fiber.Ctx) error {
	incidentID, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	photoID, err := strconv.ParseInt(ctx.Params("photoId"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid photoId")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	photo, err := r.i.DeletePhoto(ctx.UserContext(), requester.UserID, requester.IsAdmin, incidentID, photoID)
	if err != nil {
		r.l.Error(err, "restapi - v1 - deleteIncidentPhoto")
		return incidentErrorResponse(ctx, err)
	}

	storage, storageErr := objectstorage.NewFromEnv(ctx.UserContext())
	if storageErr == nil && photo.FileKey != "" {
		if err = storage.Delete(ctx.UserContext(), photo.FileKey); err != nil {
			r.l.Error(err, "restapi - v1 - deleteIncidentPhoto - DeleteObject")
		}
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Скачать документ обращения
// @Description Возвращает HTML-документ обращения по инциденту с Content-Disposition attachment. Доступно только автору или администратору.
// @ID          download-incident-document
// @Tags        incidents
// @Accept      json
// @Produce     text/html
// @Param       id path int true "ID инцидента"
// @Security    BearerAuth
// @Success     200 {string} string "HTML документ обращения"
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id}/document/download [get]
func (r *IncidentsV1) downloadIncidentDocument(ctx *fiber.Ctx) error {
	return r.sendIncidentDocument(ctx, true)
}

// @Summary     Получить печатную версию документа обращения
// @Description Возвращает HTML-документ обращения с inline Content-Disposition для печати на клиенте. Доступно только автору или администратору.
// @ID          print-incident-document
// @Tags        incidents
// @Accept      json
// @Produce     text/html
// @Param       id path int true "ID инцидента"
// @Security    BearerAuth
// @Success     200 {string} string "HTML документ обращения"
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id}/document/print [get]
func (r *IncidentsV1) printIncidentDocument(ctx *fiber.Ctx) error {
	return r.sendIncidentDocument(ctx, false)
}

func (r *IncidentsV1) sendIncidentDocument(ctx *fiber.Ctx, asAttachment bool) error {
	incidentID, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	doc, err := r.i.RenderDocument(ctx.UserContext(), requester.UserID, requester.IsAdmin, incidentID)
	if err != nil {
		r.l.Error(err, "restapi - v1 - sendIncidentDocument")
		return incidentErrorResponse(ctx, err)
	}

	ctx.Set(fiber.HeaderContentType, doc.ContentType)
	if asAttachment {
		ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, doc.FileName))
	} else {
		ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`inline; filename="%s"`, doc.FileName))
	}

	return ctx.Status(http.StatusOK).SendString(doc.BodyHTML)
}

// @Summary     Отправить документ обращения на email
// @Description Отправляет документ обращения на указанный email. Если email не передан, используется email автора инцидента. Доступно только автору или администратору.
// @ID          email-incident-document
// @Tags        incidents
// @Accept      json
// @Produce     json
// @Param       id path int true "ID инцидента"
// @Param       request body request.SendIncidentDocumentEmail false "Email получателя; можно не передавать"
// @Security    BearerAuth
// @Success     200 {object} response.MessageResponse
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /incidents/{id}/document/email [post]
func (r *IncidentsV1) emailIncidentDocument(ctx *fiber.Ctx) error {
	incidentID, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return errorResponse(ctx, http.StatusBadRequest, "invalid id")
	}
	requester, err := requireRequester(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	var body request.SendIncidentDocumentEmail
	if len(ctx.Body()) > 0 {
		if err = ctx.BodyParser(&body); err != nil {
			return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
		}
		if err = r.v.Struct(body); err != nil {
			return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
		}
	}

	if err = r.i.SendDocumentByEmail(ctx.UserContext(), requester.UserID, requester.IsAdmin, incidentID, body.Email); err != nil {
		r.l.Error(err, "restapi - v1 - emailIncidentDocument")
		return incidentErrorResponse(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(response.MessageResponse{Message: "document sent"})
}

func toIncidentResponse(incident entity.Incident) response.Incident {
	photos := make([]response.IncidentPhoto, 0, len(incident.Photos))
	for _, photo := range incident.Photos {
		photos = append(photos, toIncidentPhotoResponse(photo))
	}

	return response.Incident{
		ID:             incident.ID,
		UserID:         incident.UserID,
		CategoryID:     incident.CategoryID,
		CategoryTitle:  incident.CategoryTitle,
		Title:          incident.Title,
		Description:    incident.Description,
		Status:         incident.Status,
		DepartmentName: incident.DepartmentName,
		City:           incident.City,
		Street:         incident.Street,
		House:          incident.House,
		AddressText:    incident.AddressText,
		Latitude:       incident.Latitude,
		Longitude:      incident.Longitude,
		Photos:         photos,
		PublishedAt:    incident.PublishedAt,
		CreatedAt:      incident.CreatedAt,
		UpdatedAt:      incident.UpdatedAt,
	}
}

func toIncidentResponses(incidents []entity.Incident) []response.Incident {
	result := make([]response.Incident, 0, len(incidents))
	for _, incident := range incidents {
		result = append(result, toIncidentResponse(incident))
	}
	return result
}

func toIncidentPhotoResponse(photo entity.IncidentPhoto) response.IncidentPhoto {
	return response.IncidentPhoto{
		ID:          photo.ID,
		FileURL:     photo.FileURL,
		ContentType: photo.ContentType,
		SizeBytes:   photo.SizeBytes,
		SortOrder:   photo.SortOrder,
		CreatedAt:   photo.CreatedAt,
	}
}
