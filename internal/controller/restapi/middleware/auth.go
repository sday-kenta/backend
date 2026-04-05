package middleware

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/pkg/authjwt"
)

const authUserKey = "auth_user"

type AuthUser struct {
	UserID int64
	Role   string
}

func (u AuthUser) IsAdmin() bool {
	return u.Role == entity.UserRoleAdmin
}

func OptionalAuthJWT(jwtManager *authjwt.Manager) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		header := strings.TrimSpace(ctx.Get(fiber.HeaderAuthorization))
		if header == "" {
			return ctx.Next()
		}

		token, ok := extractBearerToken(header)
		if !ok {
			return authErrorResponse(ctx, http.StatusUnauthorized, "invalid authorization header")
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			return authErrorResponse(ctx, http.StatusUnauthorized, "invalid or expired token")
		}

		ctx.Locals(authUserKey, AuthUser{
			UserID: claims.UserID,
			Role:   claims.Role,
		})

		return ctx.Next()
	}
}

func RequireAuth() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if _, ok := CurrentUser(ctx); !ok {
			return authErrorResponse(ctx, http.StatusUnauthorized, "authentication required")
		}
		return ctx.Next()
	}
}

func RequireAdmin() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user, ok := CurrentUser(ctx)
		if !ok {
			return authErrorResponse(ctx, http.StatusUnauthorized, "authentication required")
		}
		if !user.IsAdmin() {
			return authErrorResponse(ctx, http.StatusForbidden, "access denied")
		}
		return ctx.Next()
	}
}

func CurrentUser(ctx *fiber.Ctx) (AuthUser, bool) {
	user, ok := ctx.Locals(authUserKey).(AuthUser)
	return user, ok
}

func extractBearerToken(header string) (string, bool) {
	if len(header) < len("Bearer ")+1 {
		return "", false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func authErrorResponse(ctx *fiber.Ctx, code int, msg string) error {
	return ctx.Status(code).JSON(fiber.Map{"error": msg})
}
