package server

import (
	"context"
	"fmt"
	"github.com/fekalegi/multi-tenant-system/api/handler"
	"github.com/fekalegi/multi-tenant-system/config"
	"github.com/fekalegi/multi-tenant-system/internal/auth"
	"github.com/fekalegi/multi-tenant-system/internal/message"
	"github.com/fekalegi/multi-tenant-system/internal/tenant"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	_ "github.com/fekalegi/multi-tenant-system/docs"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type Server struct {
	e    *echo.Echo
	port int
	log  zerolog.Logger
}

func NewServer(cfg *config.Config, manager *tenant.Manager, messageService *message.Service, jwtManager *auth.JWTManager, log zerolog.Logger) *Server {
	e := echo.New()
	registerRoutes(e, manager, messageService, jwtManager)

	return &Server{
		e:    e,
		port: cfg.Server.Port,
		log:  log,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.log.Info().Str("addr", addr).Msg("Starting HTTP server")
	return s.e.Start(addr)
}

func (s *Server) Stop(ctx context.Context) error {
	s.log.Info().Msg("Shutting down HTTP server")
	return s.e.Shutdown(ctx)
}

func registerRoutes(e *echo.Echo, manager *tenant.Manager, messageService *message.Service, jwtManager *auth.JWTManager) {

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	public := e.Group("/api")
	loginHandler := handler.NewLoginHandler(jwtManager)
	loginHandler.RegisterRoutes(public)

	protected := e.Group("/api", JWTAuthMiddleware(jwtManager))
	tenantHandler := handler.NewTenantHandler(manager)
	tenantHandler.RegisterTenantRoutes(protected)

	messageHandler := handler.NewMessageHandler(messageService)
	messageHandler.RegisterMessageRoute(protected)
}

func (s *Server) GetEcho() *echo.Echo {
	return s.e
}
