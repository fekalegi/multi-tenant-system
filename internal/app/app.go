package app

import (
	"context"
	"errors"
	"github.com/fekalegi/multi-tenant-system/internal/message"
	message2 "github.com/fekalegi/multi-tenant-system/internal/repository/postgresql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fekalegi/multi-tenant-system/config"
	"github.com/fekalegi/multi-tenant-system/db"
	"github.com/fekalegi/multi-tenant-system/internal/rabbitmq"
	"github.com/fekalegi/multi-tenant-system/internal/server"
	"github.com/fekalegi/multi-tenant-system/internal/tenant"
	"github.com/fekalegi/multi-tenant-system/pkg/logger"
)

func Start(cfg *config.Config) {
	// Logger
	log := logger.New()

	// DB
	dbPool := db.NewPostgres(cfg.Database.URL, log)

	// Migrate
	err := db.RunMigrations(dbPool)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// RabbitMQ
	rmq := rabbitmq.NewConnection(cfg.RabbitMQ.URL, log)

	// TenantManager
	manager := tenant.NewTenantService(rmq, dbPool, log, cfg.Workers)

	// Publisher
	publisher := rabbitmq.NewPublisher(rmq, log)

	// Message Service
	messageRepo := message2.NewMessageRepository(dbPool)
	messageService := message.NewService(publisher, messageRepo)

	// HTTP Server
	srv := server.NewServer(cfg, manager, messageService, log)

	// Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		// We check for http.ErrServerClosed and treat it as a clean exit, not a fatal error.
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("Shutting down...")

	// --- Begin Clean Shutdown Sequence ---
	log.Info().Msg("Shutdown signal received. Starting graceful shutdown...")

	// Create a timeout context for the entire shutdown process
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Stop the HTTP server
	if err := srv.Stop(ctxTimeout); err != nil {
		log.Warn().Err(err).Msg("HTTP server shutdown error")
	}
	log.Info().Msg("HTTP server stopped")

	// 2. Stop the tenant consumers using the new function
	manager.ShutdownConsumers(ctxTimeout) // Pass the timeout context
	log.Info().Msg("Tenant consumers stopped")

	// 3. Close connections
	rmq.Close()
	dbPool.Close()
	log.Info().Msg("Connections closed. Shutdown complete.")
}
