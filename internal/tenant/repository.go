package tenant

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for partition management.
type Repository interface {
	CreatePartitionForTenant(ctx context.Context, tenantID string) error
	DeletePartitionForTenant(ctx context.Context, tenantID string) error
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

func (r *repository) CreatePartitionForTenant(ctx context.Context, tenantID string) error {
	partitionName := fmt.Sprintf("messages_tenant_%s", strings.ReplaceAll(tenantID, "-", "_"))

	createPartitionSQL := fmt.Sprintf(
		"CREATE TABLE %s PARTITION OF messages FOR VALUES IN ('%s')",
		pgx.Identifier{partitionName}.Sanitize(), // Safely quote the table name
		tenantID,                                 // Embed the tenantID as a literal
	)

	_, err := r.db.Exec(ctx, createPartitionSQL)
	if err != nil {
		return fmt.Errorf("could not create message partition for tenant %s: %w", tenantID, err)
	}
	return nil
}

func (r *repository) DeletePartitionForTenant(ctx context.Context, tenantID string) error {
	partitionName := fmt.Sprintf("messages_tenant_%s", strings.ReplaceAll(tenantID, "-", "_"))

	dropPartitionSQL := fmt.Sprintf(
		"DROP TABLE %s",
		pgx.Identifier{partitionName}.Sanitize(),
	)

	_, err := r.db.Exec(ctx, dropPartitionSQL)
	if err != nil {
		return fmt.Errorf("could not drop message partition for tenant %s: %w", tenantID, err)
	}
	return nil
}
