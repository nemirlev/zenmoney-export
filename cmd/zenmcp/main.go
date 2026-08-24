package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nemirlev/zenmoney-export/v2/config"
	"github.com/nemirlev/zenmoney-export/v2/internal/analytics"
	"github.com/nemirlev/zenmoney-export/v2/internal/db/postgres"
	"github.com/nemirlev/zenmoney-export/v2/internal/mcpserver"
)

const (
	serverName      = "zenmcp"
	serverVersion   = "dev"
	shutdownTimeout = 10 * time.Second
	readTimeout     = 15 * time.Second
	idleTimeout     = 60 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configValue, err := config.NewMCPConfigFromEnv()
	if err != nil {
		return fmt.Errorf("load zenmcp configuration: %w", err)
	}
	logger := config.NewMCPLogger(configValue)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := postgres.NewPostgresAnalyticsStore(ctx, configValue.DatabaseURL)
	if err != nil {
		return fmt.Errorf("initialize analytics database: %w", err)
	}
	defer func() {
		if closeErr := store.Close(context.Background()); closeErr != nil {
			logger.Error("close analytics database", "error", closeErr)
		}
	}()

	service, err := analytics.NewService(store, analytics.Limits{
		MaxPeriodDays: configValue.MaxPeriodDays, DefaultPageSize: configValue.DefaultPageSize,
		MaxPageSize: configValue.MaxPageSize, MaxChartPoints: configValue.MaxChartPoints,
		MaxFilterValues: configValue.MaxFilterValues, StaleAfter: configValue.StaleAfter,
		DefaultTimezone: configValue.ReportTimezone,
	})
	if err != nil {
		return fmt.Errorf("initialize analytics service: %w", err)
	}
	server, err := mcpserver.New(service, mcpserver.ServerOptions{
		Name: serverName, Version: serverVersion,
	})
	if err != nil {
		return fmt.Errorf("initialize MCP server: %w", err)
	}

	originProtection := http.NewCrossOriginProtection()
	for _, origin := range configValue.AllowedOrigins {
		if err := originProtection.AddTrustedOrigin(origin); err != nil {
			return fmt.Errorf("configure trusted MCP origin: %w", err)
		}
	}
	identityResolver, err := buildIdentityResolver(configValue)
	if err != nil {
		return fmt.Errorf("initialize MCP authentication: %w", err)
	}
	handlers, err := mcpserver.NewHTTPHandlers(server, mcpserver.HTTPOptions{
		IdentityResolver: identityResolver,
		ReadinessCheck: func(request *http.Request) error {
			return store.Ping(request.Context())
		},
		ProtectOrigin:                originProtection.Handler,
		JSONResponse:                 true,
		MaxRequestBodyBytes:          configValue.MaxRequestBodyBytes,
		PropagateRequestCancellation: true,
		RequestTimeout:               configValue.RequestTimeout,
	})
	if err != nil {
		return fmt.Errorf("initialize MCP HTTP handlers: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(configValue.Endpoint, handlers.MCP)
	mux.Handle("/healthz", handlers.Health)
	mux.Handle("/readyz", handlers.Readiness)
	httpServer := &http.Server{
		Addr: configValue.ListenAddress, Handler: mux,
		ReadHeaderTimeout: readTimeout, ReadTimeout: readTimeout, IdleTimeout: idleTimeout,
	}

	serveError := make(chan error, 1)
	go func() {
		serveError <- httpServer.ListenAndServe()
	}()
	logger.Info(
		"zenmcp listening",
		"address", configValue.ListenAddress,
		"endpoint", configValue.Endpoint,
		"auth_mode", configValue.AuthMode,
	)

	select {
	case err := <-serveError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve zenmcp: %w", err)
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown zenmcp: %w", err)
		}
		return nil
	}
}

func buildIdentityResolver(configValue *config.MCPConfig) (mcpserver.IdentityResolver, error) {
	principal := analytics.Principal{
		Subject:  "local-development",
		AllUsers: len(configValue.UserIDs) == 0,
		UserIDs:  append([]int64(nil), configValue.UserIDs...),
	}
	if configValue.AuthMode == config.MCPAuthLocal {
		return mcpserver.StaticIdentityResolver{Principal: principal}, nil
	}

	principal.Subject = "configured-bearer"
	return mcpserver.NewBearerIdentityResolver(configValue.BearerToken, principal)
}
