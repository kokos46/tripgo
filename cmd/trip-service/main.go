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

	"github.com/exaring/otelpgx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kokos46/tripgo/internal/config"
	domaintrip "github.com/kokos46/tripgo/internal/domain/trip"
	api "github.com/kokos46/tripgo/internal/generated"
	httpmiddleware "github.com/kokos46/tripgo/internal/http/middleware"
	triphttp "github.com/kokos46/tripgo/internal/http/trip"
	"github.com/kokos46/tripgo/internal/infra/postgres/transaction"
	"github.com/kokos46/tripgo/internal/infra/telemetry"
	triprepo "github.com/kokos46/tripgo/internal/repository/trip"
	"github.com/kokos46/tripgo/internal/usecase/trip/create"
	"github.com/kokos46/tripgo/internal/usecase/trip/finish"
	"github.com/kokos46/tripgo/internal/usecase/trip/get"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	providers, err := telemetry.Setup(ctx)
	if err != nil {
		return fmt.Errorf("setup telemetry: %w", err)
	}
	providersShutdown := false
	defer func() {
		if providersShutdown {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if shutdownErr := providers.Shutdown(shutdownCtx); shutdownErr != nil {
			slog.Error("shutdown telemetry", "error", shutdownErr)
		}
	}()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("parse PostgreSQL config: %w", err)
	}
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()
	poolConfig.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns
	poolConfig.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	pingCtx, cancelPing := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}

	repository := triprepo.NewRepository(pool, cfg.DatabaseQueryTimeout)
	txManager := &transaction.Manager{Pool: pool, QueryTimeout: cfg.DatabaseQueryTimeout}
	app := triphttp.New(
		create.New(txManager, repository),
		get.New(repository),
		finish.New(txManager, repository),
		func() error {
			pingCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
			defer cancel()
			return pool.Ping(pingCtx)
		},
	)

	router := chi.NewRouter()
	router.Use(httpmiddleware.Observability(providers.Tracer, providers.Meter, slog.Default()))
	server := api.HandlerWithOptions(app, api.ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			triphttp.WriteProblem(w, r, &domaintrip.ValidationError{Field: "request", Reason: "is invalid"})
		},
	})
	server = httpmiddleware.Recover(slog.Default())(server)
	server = otelhttp.NewHandler(server, "http.server")

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("HTTP server is listening", "address", cfg.HTTPAddr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		serverShutdownErr := httpServer.Shutdown(shutdownCtx)
		if serverShutdownErr != nil {
			slog.Error("graceful shutdown exceeded its deadline; closing active connections", "error", serverShutdownErr)
			if closeErr := httpServer.Close(); closeErr != nil {
				serverShutdownErr = errors.Join(serverShutdownErr, closeErr)
			}
		}
		telemetryShutdownErr := providers.Shutdown(shutdownCtx)
		providersShutdown = true
		if serverShutdownErr != nil || telemetryShutdownErr != nil {
			return errors.Join(
				wrapError("shutdown HTTP server", serverShutdownErr),
				wrapError("shutdown telemetry", telemetryShutdownErr),
			)
		}
		return nil
	}
}

func wrapError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
