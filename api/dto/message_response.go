package dto

import (
	"github.com/fekalegi/multi-tenant-system/internal/domain"
)

type GetMessagesResponse struct {
	Data       []*domain.Message `json:"data"`
	NextCursor string            `json:"next_cursor,omitempty" example:"eyJpZCI6ImYx...YjAifQ=="`
}
