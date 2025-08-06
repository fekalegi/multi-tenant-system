package server

import (
	"net/http"
	"strings"

	"github.com/fekalegi/multi-tenant-system/internal/auth"
	"github.com/labstack/echo/v4"
)

const (
	ContextUserIDKey   = "user_id"
	ContextTenantIDKey = "tenant_id"
)

func JWTAuthMiddleware(jwtManager *auth.JWTManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing or invalid token"})
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwtManager.Parse(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid token"})
			}

			// Set tenant and user in context
			c.Set(ContextUserIDKey, claims.UserID)
			c.Set(ContextTenantIDKey, claims.TenantID)

			return next(c)
		}
	}
}
