package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func NewPostgres(dsn string, log zerolog.Logger) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid DB config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to connect to DB")
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("Ping to DB failed")
	}

	log.Info().Msg("Connected to PostgreSQL")
	return pool
}
