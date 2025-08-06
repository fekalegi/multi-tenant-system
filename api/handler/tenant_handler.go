package handler

import (
	"github.com/fekalegi/multi-tenant-system/internal/domain"
	"net/http"

	"github.com/fekalegi/multi-tenant-system/api/dto" // Make sure to import the dto package
	"github.com/fekalegi/multi-tenant-system/internal/tenant"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// TenantHandler handles tenant operations
type TenantHandler struct {
	manager *tenant.Manager
}

// NewTenantHandler creates a new TenantHandler instance
func NewTenantHandler(m *tenant.Manager) *TenantHandler {
	return &TenantHandler{manager: m}
}

// RegisterTenantRoutes registers tenant-related HTTP routes
func (h *TenantHandler) RegisterTenantRoutes(e *echo.Group) {
	e.POST("/tenants", h.CreateTenant)
	e.DELETE("/tenants/:id", h.DeleteTenant)
	e.PUT("/tenants/:id/config/concurrency", h.UpdateConcurrency)
}

// CreateTenant godoc
// @Summary Create a new tenant
// @Description Creates a new tenant and returns its generated ID and name.
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body dto.CreateTenantRequest true "Tenant name"
// @Success 201 {object} dto.CreateTenantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/tenants [post]
func (h *TenantHandler) CreateTenant(c echo.Context) error {
	var req dto.CreateTenantRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
	}
	id := uuid.New().String()

	if err := h.manager.CreateTenant(c.Request().Context(), id, req.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	// Use the new response struct
	response := dto.CreateTenantResponse{
		ID:   id,
		Name: req.Name,
	}
	return c.JSON(http.StatusCreated, response)
}

// DeleteTenant godoc
// @Summary Delete a tenant
// @Description Deletes a tenant by its ID.
// @Tags tenants
// @Param id path string true "Tenant ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c echo.Context) error {
	id := c.Param("id")
	if err := h.manager.DeleteTenant(c.Request().Context(), id); err != nil {
		// Use the standard error response struct
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateConcurrency godoc
// @Summary Update tenant concurrency setting
// @Description Updates the number of concurrent workers for a specific tenant.
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body tenant.ConcurrencyConfig true "Concurrency config"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/tenants/{id}/config/concurrency [put]
func (h *TenantHandler) UpdateConcurrency(c echo.Context) error {
	id := c.Param("id")

	var req domain.ConcurrencyConfig
	if err := c.Bind(&req); err != nil || req.Workers <= 0 {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request: 'workers' must be a positive number"})
	}

	if err := h.manager.UpdateConcurrency(id, req.Workers); err != nil {
		return c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
	}

	// Use the new message response struct
	response := dto.MessageResponse{
		Message: "concurrency updated successfully",
	}
	return c.JSON(http.StatusOK, response)
}
