package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fekalegi/multi-tenant-system/internal/rabbitmq"
	"github.com/google/uuid"
)

type Service struct {
	publisher  *rabbitmq.Publisher
	repository Repository
}

func NewService(publisher *rabbitmq.Publisher, repo Repository) *Service {
	return &Service{
		publisher:  publisher,
		repository: repo,
	}
}

func (s *Service) PublishMessage(ctx context.Context, tenantID uuid.UUID, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &Message{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Payload:   body,
		CreatedAt: time.Now(),
	}

	// Store in database first (can be swapped order if needed)
	if err := s.repository.InsertMessage(ctx, msg); err != nil {
		return err
	}

	// Publish to RabbitMQ
	if err := s.publisher.PublishToTenantQueue(tenantID.String(), body); err != nil {
		return fmt.Errorf("rabbitmq publish error: %w", err)
	}

	return nil
}

func (s *Service) FetchMessagesWithCursor(ctx context.Context, cursor string, limit int) ([]*Message, string, error) {
	if limit <= 0 {
		return nil, "", errors.New("limit must be > 0")
	}
	return s.repository.GetMessagesWithCursor(ctx, cursor, limit)
}
