package handler

import (
	"github.com/google/uuid"
	"net/http"
	"strconv"

	"github.com/fekalegi/multi-tenant-system/api/dto" // Import the DTO package
	"github.com/fekalegi/multi-tenant-system/internal/message"
	"github.com/labstack/echo/v4"
)

// Handler for message-related endpoints.
// Note: Renaming from 'Handler' to 'MessageHandler' avoids ambiguity.
type MessageHandler struct {
	messageService *message.Service
}

// NewMessageHandler initializes the Handler for message-related endpoints
func NewMessageHandler(messageService *message.Service) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// RegisterMessageRoute registers message-related routes to the Echo router
func (h *MessageHandler) RegisterMessageRoute(e *echo.Group) {
	e.POST("/messages/:tenant_id", h.Publish)
	e.GET("/messages", h.GetMessages)
}

// Publish godoc
// @Summary     Publish a message to a tenant
// @Description Publishes a JSON payload to a specific tenant's queue.
// @Tags        messages
// @Accept      json
// @Produce     json
// @Param       tenant_id path string true "Tenant ID"
// @Param       message body object true "Message Payload" example({"key": "value", "priority": 1})
// @Success     200 {object} dto.MessageResponse
// @Failure     400 {object} dto.ErrorResponse
// @Failure     500 {object} dto.ErrorResponse
// @Security 	BearerAuth
// @Router      /api/messages/{tenant_id} [post]
func (h *MessageHandler) Publish(c echo.Context) error {
	tenantID := c.Param("tenant_id")

	var body map[string]interface{}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid json payload"})
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	}

	if err := h.messageService.PublishMessage(c.Request().Context(), tenantUUID, body); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, dto.MessageResponse{Message: "message sent successfully"})
}

// GetMessages godoc
// @Summary     Get messages with cursor-based pagination
// @Description Retrieves a paginated list of all processed messages.
// @Tags        messages
// @Produce     json
// @Param       cursor query string false "Cursor for pagination"
// @Param       limit query int false "Limit"
// @Success     200 {object} dto.GetMessagesResponse
// @Failure     500 {object} dto.ErrorResponse
// @Security 	BearerAuth
// @Router      /api/messages [get]
func (h *MessageHandler) GetMessages(c echo.Context) error {
	ctx := c.Request().Context()
	cursor := c.QueryParam("cursor")
	limitStr := c.QueryParam("limit")

	limit := 1
	var err error

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid 'limit' parameter. Must be an integer."})
		}
	}
	// Assuming the service returns ([]message.Message, string, error)
	messages, nextCursor, err := h.messageService.FetchMessagesWithCursor(ctx, cursor, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
	}

	// Use the new response struct
	response := dto.GetMessagesResponse{
		Data:       messages,
		NextCursor: nextCursor,
	}

	return c.JSON(http.StatusOK, response)
}
