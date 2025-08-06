package handler

import (
	"github.com/fekalegi/multi-tenant-system/api/dto"
	"github.com/fekalegi/multi-tenant-system/internal/auth"
	"github.com/labstack/echo/v4"
	"net/http"
)

type LoginHandler struct {
	jwt *auth.JWTManager
}

func NewLoginHandler(jwt *auth.JWTManager) *LoginHandler {
	return &LoginHandler{jwt: jwt}
}

func (h *LoginHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/login", h.Login)
}

// Login godoc
// @Summary Mock login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Mock user"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/login [post]
func (h *LoginHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil || req.UserID == "" || req.TenantID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	token, err := h.jwt.Generate(req.UserID, req.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to generate token"})
	}

	return c.JSON(http.StatusOK, dto.LoginResponse{Token: token})
}
