package dto

import (
	"github.com/fekalegi/multi-tenant-system/internal/message"
)

type GetMessagesResponse struct {
	Data       []*message.Message `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty" example:"eyJpZCI6ImYx...YjAifQ=="`
}
