//go:build integration

package tenant_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	// --- Your Project's Packages ---
	"github.com/fekalegi/multi-tenant-system/config"
	"github.com/fekalegi/multi-tenant-system/db"
	"github.com/fekalegi/multi-tenant-system/internal/message"
	"github.com/fekalegi/multi-tenant-system/internal/rabbitmq"
	"github.com/fekalegi/multi-tenant-system/internal/server"
	"github.com/fekalegi/multi-tenant-system/internal/tenant"
	"github.com/fekalegi/multi-tenant-system/pkg/logger" // Adjusted path based on your structure

	// --- External Dependencies ---
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	echoServer *echo.Echo
	dbPool     *pgxpool.Pool
	tenantID   string
	log        zerolog.Logger
}

// SetupSuite runs once before all tests in the suite to set up the environment.
func (s *IntegrationTestSuite) SetupSuite() {
	s.log = logger.New() // Use your project's logger
	pool, err := dockertest.NewPool("")
	require.NoError(s.T(), err, "Could not construct docker pool")

	// --- Start PostgreSQL Container ---
	pgResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres", Tag: "13",
		Env: []string{"POSTGRES_USER=testuser", "POSTGRES_PASSWORD=testpassword", "POSTGRES_DB=testdb"},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(s.T(), err, "Could not start PostgreSQL resource")

	// --- Start RabbitMQ Container ---
	rmqResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq", Tag: "3-management",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(s.T(), err, "Could not start RabbitMQ resource")

	s.T().Cleanup(func() {
		s.log.Info().Msg("Purging test containers...")
		require.NoError(s.T(), pool.Purge(pgResource))
		require.NoError(s.T(), pool.Purge(rmqResource))
	})

	// --- Connect to Dependencies ---
	pgHostAndPort := pgResource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://testuser:testpassword@%s/testdb?sslmode=disable", pgHostAndPort)

	rmqHostAndPort := rmqResource.GetHostPort("5672")
	rabbitmqURL := fmt.Sprintf("amqp://guest:guest@%s/", rmqHostAndPort)

	require.NoError(s.T(), pool.Retry(func() error {
		var err error
		s.dbPool, err = pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			return err
		}
		return s.dbPool.Ping(context.Background())
	}), "Could not connect to PostgreSQL")

	// --- Run Migrations ---
	require.NoError(s.T(), db.RunMigrations(s.dbPool), "Could not run migrations")

	// --- Assemble the Application Stack (mirroring your main.go) ---
	cfg := &config.Config{ /* Populate if needed */ }
	rmqConn := rabbitmq.NewConnection(rabbitmqURL, s.log)

	tenantManager := tenant.NewTenantService(rmqConn, s.dbPool, s.log, 3) // Using your constructor

	publisher := rabbitmq.NewPublisher(rmqConn, s.log)
	messageRepo := message.NewRepository(s.dbPool)
	messageService := message.NewService(publisher, messageRepo)

	srv := server.NewServer(cfg, tenantManager, messageService, s.log)
	s.echoServer = srv.GetEcho()
}

// TearDownSuite runs once after all tests in the suite.
func (s *IntegrationTestSuite) TearDownSuite() {
	s.dbPool.Close()
}

// TestTenantLifecycle runs the main test sequence.
func (s *IntegrationTestSuite) TestTenantLifecycle() {
	s.Run("1_When_CreateTenantIsCalled_Then_PartitionAndQueueAreCreated", s.testCreateTenant)
	s.Run("2_When_MessageIsPublished_Then_ItIsConsumedAndStored", s.testPublishAndConsumeMessage)
	s.Run("3_When_DeleteTenantIsCalled_Then_PartitionIsDropped", s.testDeleteTenant)
}

func (s *IntegrationTestSuite) testCreateTenant() {
	body := bytes.NewBufferString(`{"name": "integration-test-tenant"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tenants", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.echoServer.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusCreated, rec.Code, "Expected status 201 Created")

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(s.T(), err)
	s.tenantID = resp["id"]
	require.NotEmpty(s.T(), s.tenantID, "Tenant ID should be returned in the response")

	// Verification: Check if the database partition was created
	partitionExists, err := s.checkPartitionExists(s.tenantID)
	require.NoError(s.T(), err)
	require.True(s.T(), partitionExists, "Database partition for the new tenant should exist")
}

func (s *IntegrationTestSuite) testPublishAndConsumeMessage() {
	require.NotEmpty(s.T(), s.tenantID, "testCreateTenant must run first to get a tenantID")

	msgBody := bytes.NewBufferString(`{"data": "hello from integration test"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/messages/%s", s.tenantID), msgBody)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.echoServer.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusOK, rec.Code)

	// Verification: The message consumer runs in the background. We need to poll the DB.
	var messageCount int
	require.Eventually(s.T(), func() bool {
		err := s.dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM messages WHERE tenant_id = $1", s.tenantID).Scan(&messageCount)
		return err == nil && messageCount > 0
	}, 5*time.Second, 200*time.Millisecond, "Message should be consumed and saved to the database")
}

func (s *IntegrationTestSuite) testDeleteTenant() {
	require.NotEmpty(s.T(), s.tenantID, "testCreateTenant must run first to get a tenantID")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/tenants/%s", s.tenantID), nil)
	rec := httptest.NewRecorder()

	s.echoServer.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusNoContent, rec.Code)

	// Verification: Check if the database partition was dropped
	partitionExists, err := s.checkPartitionExists(s.tenantID)
	require.NoError(s.T(), err)
	require.False(s.T(), partitionExists, "Database partition should be dropped after tenant deletion")
}

func (s *IntegrationTestSuite) checkPartitionExists(tenantID string) (bool, error) {
	expectedPartitionName := fmt.Sprintf("messages_tenant_%s", strings.ReplaceAll(tenantID, "-", "_"))
	var exists bool
	query := `SELECT EXISTS (SELECT FROM pg_class WHERE relkind = 'r' AND relname = $1);`
	err := s.dbPool.QueryRow(context.Background(), query, expectedPartitionName).Scan(&exists)
	return exists, err
}

// TestIntegrationTestSuite is the entry point for running the test suite.
func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
