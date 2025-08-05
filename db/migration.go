package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(pool *pgxpool.Pool) error {
	schema := `
CREATE TABLE IF NOT EXISTS messages (
	tenant_id UUID NOT NULL,
	id UUID NOT NULL,
	payload JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (tenant_id, id)
) PARTITION BY LIST (tenant_id);
`
	_, err := pool.Exec(context.Background(), schema)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
