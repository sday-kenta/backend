// backend/internal/controller/restapi/middleware/logger.go

package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/config"
	"github.com/sday-kenta/backend/pkg/logger"
)

func buildRequestFields(ctx *fiber.Ctx, cfg config.Log, duration time.Duration) map[string]interface{} {
	fields := map[string]interface{}{
		"ip":             ctx.IP(),
		"method":         ctx.Method(),
		"url":            ctx.OriginalURL(),
		"status":         ctx.Response().StatusCode(),
		"response_bytes": len(ctx.Response().Body()),
		"duration_ms":    float64(duration.Microseconds()) / 1000,
	}

	if len(ctx.Queries()) > 0 {
		fields["query"] = ctx.Queries()
	}

	if user, ok := CurrentUser(ctx); ok {
		fields["user_id"] = user.UserID
		fields["role"] = user.Role
	}

	if cfg.HTTPLogHeaders {
		if headers := sanitizeHeaders(ctx.GetReqHeaders()); len(headers) > 0 {
			fields["headers"] = headers
		}
	}

	if cfg.HTTPLogBody {
		if body := sanitizeRequestBody(ctx, cfg.HTTPLogBodyMaxBytes); body != nil {
			fields["body"] = body
		}
	}

	return fields
}

func Logger(l logger.Interface, cfg config.Log) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		startedAt := time.Now()
		err := ctx.Next()

		l.InfoFields("http request", buildRequestFields(ctx, cfg, time.Since(startedAt)))

		return err
	}
}

func sanitizeHeaders(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}

	sanitized := make(map[string][]string, len(headers))
	for key, values := range headers {
		if isSensitiveHeader(key) {
			sanitized[key] = []string{"<redacted>"}
			continue
		}

		copied := make([]string, 0, len(values))
		for _, value := range values {
			copied = append(copied, value)
		}
		sanitized[key] = copied
	}

	return sanitized
}

func sanitizeRequestBody(ctx *fiber.Ctx, maxBytes int) interface{} {
	contentType := strings.ToLower(strings.TrimSpace(ctx.Get(fiber.HeaderContentType)))
	if !strings.Contains(contentType, "json") {
		return nil
	}

	body := bytesWithLimit(ctx.Body(), maxBytes)
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body)
	}

	redactJSON(payload)
	return payload
}

func bytesWithLimit(src []byte, maxBytes int) []byte {
	if maxBytes <= 0 || len(src) <= maxBytes {
		return src
	}

	truncated := make([]byte, 0, maxBytes+16)
	truncated = append(truncated, src[:maxBytes]...)
	truncated = append(truncated, []byte("...(truncated)")...)
	return truncated
}

func redactJSON(value interface{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, nested := range typed {
			if isSensitiveField(key) {
				typed[key] = "<redacted>"
				continue
			}
			redactJSON(nested)
		}
	case []interface{}:
		for _, nested := range typed {
			redactJSON(nested)
		}
	}
}

func isSensitiveHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "authorization", "cookie", "set-cookie", "x-api-key":
		return true
	default:
		return false
	}
}

func isSensitiveField(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "password", "password_hash", "old_password", "new_password", "confirm_password",
		"code", "token", "access_token", "refresh_token", "authorization",
		"email", "phone", "telephone", "mobile", "login", "username", "identifier",
		"first_name", "last_name", "middle_name", "name", "full_name",
		"city", "street", "house", "apartment", "building", "address_text", "postcode", "postal_code", "zip",
		"fcm_token":
		return true
	default:
		return false
	}
}
