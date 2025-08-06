package message

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	InsertMessage(ctx context.Context, msg *Message) error
	GetMessagesWithCursor(ctx context.Context, cursor string, limit int) ([]*Message, string, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

func (r *repository) InsertMessage(ctx context.Context, msg *Message) error {
	payloadJSON, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO messages (id, tenant_id, payload, created_at)
		VALUES ($1, $2, $3, $4)
	`, msg.ID, msg.TenantID, payloadJSON, msg.CreatedAt)

	return err
}

func (r *repository) GetMessagesWithCursor(ctx context.Context, cursor string, limit int) ([]*Message, string, error) {
	var afterTime time.Time
	var afterID uuid.UUID

	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: not base64 encoded")
		}

		parts := strings.SplitN(string(decoded), "|", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor: malformed structure")
		}

		afterTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: could not parse time")
		}

		afterID, err = uuid.Parse(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: could not parse id")
		}
	}

	query := `
		SELECT id, tenant_id, payload, created_at
		FROM messages
		WHERE ($1 = '' OR (created_at, id) > ($2, $3))
		ORDER BY created_at, id
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, cursor, afterTime, afterID, limit)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	messages := []*Message{}
	var nextCursor string

	for rows.Next() {
		var (
			m       Message
			rawJSON []byte
		)
		err := rows.Scan(&m.ID, &m.TenantID, &rawJSON, &m.CreatedAt)
		if err != nil {
			return nil, "", err
		}

		_ = json.Unmarshal(rawJSON, &m.Payload)
		messages = append(messages, &m)
	}

	if len(messages) == limit {
		last := messages[len(messages)-1]
		rawCursor := fmt.Sprintf("%s|%s", last.CreatedAt.Format(time.RFC3339Nano), last.ID)
		nextCursor = base64.StdEncoding.EncodeToString([]byte(rawCursor))
	}

	return messages, nextCursor, nil
}
