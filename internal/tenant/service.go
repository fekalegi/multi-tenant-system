package tenant

import (
	"context"
	"fmt"
	"github.com/fekalegi/multi-tenant-system/internal/domain"
	message2 "github.com/fekalegi/multi-tenant-system/internal/repository/postgresql"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fekalegi/multi-tenant-system/internal/rabbitmq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Manager struct {
	mu         sync.RWMutex
	consumers  map[string]*tenantConsumer
	Rmq        *rabbitmq.Connection
	db         *pgxpool.Pool
	Log        zerolog.Logger
	defaultWkr int
	tenantRepo message2.TenantRepository
	msgRepo    message2.MessageRepository
}

type tenantConsumer struct {
	cancelFunc context.CancelFunc
	workers    int
}

func NewTenantService(rmq *rabbitmq.Connection, db *pgxpool.Pool, log zerolog.Logger, defaultWkr int) *Manager {
	return &Manager{
		consumers:  make(map[string]*tenantConsumer),
		Rmq:        rmq,
		db:         db,
		Log:        log,
		defaultWkr: defaultWkr,
		msgRepo:    message2.NewMessageRepository(db),
		tenantRepo: message2.NewTenantRepository(db),
	}
}

func (m *Manager) CreateTenant(ctx context.Context, id string, name string) error {
	err := m.tenantRepo.CreatePartitionForTenant(ctx, id)
	if err != nil {
		m.Log.Error().Err(err).Str("tenant_id", id).Msg("Failed to create tenant")
		return err
	}

	// Queue name
	queueName := fmt.Sprintf("tenant_%s_queue", id)

	// Create queue
	ch, err := m.Rmq.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare failed: %w", err)
	}

	// Context for cancel
	ctxConsumer, cancel := context.WithCancel(context.Background())

	// Start consumer goroutine
	go m.startConsumer(ctxConsumer, id, queueName, m.defaultWkr)

	// Track tenant
	m.consumers[id] = &tenantConsumer{
		cancelFunc: cancel,
		workers:    m.defaultWkr,
	}
	m.Log.Info().Str("tenant_id", id).Str("name", name).Msg("Tenant created and consumer started")

	return nil
}

func (m *Manager) DeleteTenant(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	consumer, ok := m.consumers[id]
	if !ok {
		return fmt.Errorf("tenant not found")
	}

	// Signal shutdown
	consumer.cancelFunc()

	// Delete queue
	ch, err := m.Rmq.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	queueName := fmt.Sprintf("tenant_%s_queue", id)
	_, err = ch.QueueDelete(queueName, false, false, false)
	if err != nil {
		return fmt.Errorf("queue delete failed: %w", err)
	}

	delete(m.consumers, id)
	m.Log.Info().Str("tenant_id", id).Msg("Tenant consumer stopped and queue deleted")

	err = m.tenantRepo.DeletePartitionForTenant(ctx, id)
	if err != nil {
		m.Log.Error().Err(err).Str("tenant_id", id).Msg("Failed to delete tenant")
		return err
	}
	m.Log.Info().Str("tenant_id", id).Msg("Partition for the tenat has dropped")
	return nil
}

func (m *Manager) UpdateConcurrency(tenantID string, newWorkerCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tc, ok := m.consumers[tenantID]
	if !ok {
		return fmt.Errorf("tenant not found")
	}

	// Replace the consumer with new config
	tc.cancelFunc()
	m.Log.Info().Str("tenant_id", tenantID).Msg("Restarting consumer with new concurrency")

	ctx, cancel := context.WithCancel(context.Background())
	m.consumers[tenantID] = &tenantConsumer{
		cancelFunc: cancel,
		workers:    newWorkerCount,
	}

	go m.startConsumer(ctx, tenantID, fmt.Sprintf("tenant_%s_queue", tenantID), newWorkerCount)
	return nil
}

func (m *Manager) startConsumer(ctx context.Context, tenantID, queue string, workers int) {
	ch, err := m.Rmq.Channel()
	if err != nil {
		m.Log.Error().Err(err).Msg("Failed to open channel")
		return
	}

	msgs, err := ch.Consume(
		queue,
		"consumer-"+tenantID,
		true,  // auto-ack
		false, // exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		m.Log.Error().Err(err).Msg("Failed to start consuming")
		return
	}

	// Worker pool
	jobs := make(chan amqp.Delivery, 100)

	// Start N workers
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			for {
				select {
				case msg := <-jobs:

					messageID := uuid.New()
					tenantUUID, _ := uuid.Parse(tenantID)

					m.Log.Info().
						Str("worker", fmt.Sprint(workerID)).
						Str("tenant_id", tenantID).
						Str("msg_id", messageID.String()).
						Msg("Processing message")

					_ = m.msgRepo.InsertMessage(ctx, &domain.Message{
						ID:        messageID,
						TenantID:  tenantUUID,
						Payload:   msg.Body,
						CreatedAt: time.Now(),
					})

				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Fan in RabbitMQ deliveries into jobs
	for {
		select {
		case <-ctx.Done():
			ch.Close()
			close(jobs)
			m.Log.Info().Str("tenant_id", tenantID).Msg("Consumer shutdown")
			return
		case msg := <-msgs:
			jobs <- msg
		}
	}
}

func (m *Manager) ListenAndShutdown(timeout time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	m.Log.Info().Msg("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup

	for tenantID, consumer := range m.consumers {
		wg.Add(1)

		go func(id string, c *tenantConsumer) {
			defer wg.Done()

			// Cancel the context
			c.cancelFunc()
			m.Log.Info().Str("tenant_id", id).Msg("Consumer cancelled")

			select {
			case <-shutdownCtx.Done():
				m.Log.Warn().Str("tenant_id", id).Msg("Shutdown context expired before cleanup")
			case <-time.After(500 * time.Millisecond):
			}
		}(tenantID, consumer)
	}

	wg.Wait()
	m.Log.Info().Msg("All tenant consumers shutdown cleanly")
}

func (m *Manager) ShutdownConsumers(ctx context.Context) {
	m.Log.Info().Msg("Shutting down all tenant consumers...")

	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup

	for tenantID, consumer := range m.consumers {
		wg.Add(1)

		go func(id string, c *tenantConsumer) {
			defer wg.Done()

			// Cancel the consumer's personal context to stop its work
			c.cancelFunc()

			m.Log.Info().Str("tenant_id", id).Msg("Consumer shutdown process initiated")

			// You can add more sophisticated waiting logic here if needed,
			// but for now, cancelling is the main action.
		}(tenantID, consumer)
	}

	// Wait for all the shutdown goroutines to finish
	wg.Wait()
	m.Log.Info().Msg("All tenant consumers have been signaled to shut down.")
}
