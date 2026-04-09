package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	authmw "github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/pusherr"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

type PushV1 struct {
	p usecase.Push
	l logger.Interface
	v *validator.Validate
}

func NewPushRoutes(apiV1Group fiber.Router, p usecase.Push, l logger.Interface) {
	r := &PushV1{
		p: p,
		l: l,
		v: validator.New(validator.WithRequiredStructEnabled()),
	}

	pushGroup := apiV1Group.Group("/push")
	{
		pushGroup.Post("/devices", authmw.RequireAuth(), r.registerDevice)
		pushGroup.Delete("/devices/:deviceId", authmw.RequireAuth(), r.deleteDevice)
	}
}

func pushErrorResponse(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, pusherr.ErrDeviceNotFound):
		return errorResponse(ctx, http.StatusNotFound, "push device not found")
	default:
		return errorResponse(ctx, http.StatusInternalServerError, "push device error")
	}
}

// @Summary     Register push device
// @Description Registers or refreshes an FCM device for the authenticated user.
// @ID          register-push-device
// @Tags        push
// @Accept      json
// @Produce     json
// @Param       request body request.RegisterPushDevice true "Push device payload"
// @Security    BearerAuth
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /push/devices [post]
func (r *PushV1) registerDevice(ctx *fiber.Ctx) error {
	requester, err := currentAuthUser(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	var body request.RegisterPushDevice
	if err = ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - registerDevice - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}
	if err = r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - registerDevice - Validate")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	if err = r.p.RegisterDevice(ctx.UserContext(), requester.UserID, entity.UpsertPushDeviceInput{
		DeviceID:   body.DeviceID,
		Platform:   body.Platform,
		FCMToken:   body.FCMToken,
		AppVersion: body.AppVersion,
	}); err != nil {
		r.l.Error(err, "restapi - v1 - registerDevice")
		return pushErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}

// @Summary     Delete push device
// @Description Deletes an FCM device by client-generated device ID for the authenticated user.
// @ID          delete-push-device
// @Tags        push
// @Accept      json
// @Produce     json
// @Param       deviceId path string true "Client device ID"
// @Security    BearerAuth
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     401 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /push/devices/{deviceId} [delete]
func (r *PushV1) deleteDevice(ctx *fiber.Ctx) error {
	requester, err := currentAuthUser(ctx)
	if err != nil {
		return errorResponse(ctx, http.StatusUnauthorized, err.Error())
	}

	deviceID := strings.TrimSpace(ctx.Params("deviceId"))
	if deviceID == "" {
		return errorResponse(ctx, http.StatusBadRequest, "deviceId is required")
	}

	if err = r.p.DeleteDevice(ctx.UserContext(), requester.UserID, deviceID); err != nil {
		r.l.Error(err, "restapi - v1 - deleteDevice")
		return pushErrorResponse(ctx, err)
	}

	return ctx.SendStatus(http.StatusNoContent)
}
