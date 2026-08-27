// Command marine-farm-environment-service is a Go web service for marine
// farm environment monitoring: farming-zone management, water-quality data
// ingestion, dissolved-oxygen/temperature anomaly warnings, aerator linkage
// and daily farming logs. It embeds a native HTML/CSS/JS frontend via
// go:embed.
package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/httpapi"
	"example.com/marine-farm-environment-service/service"
	"example.com/marine-farm-environment-service/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	if err := run(); err != nil {
		slog.Error("main: fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.FromEnv()
	configureSlog(cfg.LogLevel)
	if err := cfg.Validate(); err != nil {
		return err
	}

	st := store.NewStore(cfg.DataFile)
	if err := st.Load(); err != nil {
		return err
	}

	svc := service.New(cfg, st)
	boot := service.NewBootstrap(cfg, st)
	if err := boot.SeedIfEmpty(); err != nil {
		return err
	}
	if cfg.DataFile != "" {
		if err := st.Save(); err != nil {
			return err
		}
	}

	router := httpapi.NewRouter(cfg, st, svc, webFS)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Background sweeper goroutines stop when the context is cancelled.
	svc.StartSweepers(ctx)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("marine farm environment service listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown failed", "error", err)
		}
		if cfg.DataFile != "" {
			if err := st.Save(); err != nil {
				slog.Error("final save failed", "error", err)
			} else {
				slog.Info("persisted state", "file", cfg.DataFile)
			}
		}
		slog.Info("bye")
		return nil
	}
}

// configureSlog wires a JSON text handler whose level follows LOG_LEVEL.
func configureSlog(level string) {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lv})))
}
