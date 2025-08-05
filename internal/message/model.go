package message

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Payload   []byte
	CreatedAt time.Time
}

type CursorPaginationResult struct {
	Messages   []*Message
	NextCursor string
	HasMore    bool
}
