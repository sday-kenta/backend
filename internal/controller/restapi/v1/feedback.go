package v1

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/controller/restapi/v1/request"
	"github.com/sday-kenta/backend/pkg/logger"
	"github.com/sday-kenta/backend/pkg/mailsender"
)

type FeedbackV1 struct {
	l logger.Interface
	v *validator.Validate
}

func NewFeedbackRoutes(apiV1Group fiber.Router, l logger.Interface) {
	r := &FeedbackV1{
		l: l,
		v: validator.New(validator.WithRequiredStructEnabled()),
	}
	apiV1Group.Post("/feedback", r.sendFeedback)
}

// @Summary     Отправить обращение
// @Description Письмо уходит на служебный адрес SMTP (тот же, что для кодов на почту)
// @ID          send-feedback
// @Tags        feedback
// @Accept      json
// @Produce     json
// @Param       request body request.SendFeedback true "Текст и опционально контакты"
// @Success     204
// @Failure     400 {object} response.Error
// @Failure     503 {object} response.Error
// @Failure     500 {object} response.Error
// @Router      /feedback [post]
func (r *FeedbackV1) sendFeedback(ctx *fiber.Ctx) error {
	var body request.SendFeedback
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - sendFeedback - BodyParser")
		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}
	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - sendFeedback - validate")
		return errorResponse(ctx, http.StatusBadRequest, formatValidationError(err))
	}

	to := strings.TrimSpace(os.Getenv("SMTP_MAIL"))
	if to == "" {
		return errorResponse(ctx, http.StatusServiceUnavailable, "email service is not configured")
	}

	msg := strings.TrimSpace(body.Message)
	subject := "SdayKenta — обратная связь"
	var b strings.Builder
	b.WriteString(msg)
	b.WriteString("\n\n---\n")
	if strings.TrimSpace(body.Name) != "" {
		fmt.Fprintf(&b, "Имя: %s\n", strings.TrimSpace(body.Name))
	}
	if strings.TrimSpace(body.Email) != "" {
		fmt.Fprintf(&b, "Email для ответа: %s\n", strings.TrimSpace(body.Email))
	}

	if err := mailsender.SendMail(subject, b.String(), []string{to}); err != nil {
		r.l.Error(err, "restapi - v1 - sendFeedback - SendMail")
		return errorResponse(ctx, http.StatusInternalServerError, "failed to send feedback email")
	}

	return ctx.SendStatus(http.StatusNoContent)
}
