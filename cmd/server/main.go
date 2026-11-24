package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dummyjson/internal/api"
	"dummyjson/internal/auth"
	"dummyjson/internal/store"
)

func addrFromEnv() string {
	if value := os.Getenv("ADDR"); value != "" {
		return value
	}
	return ":8888"
}

func dataDirFromEnv() string {
	if value := os.Getenv("DATA_DIR"); value != "" {
		return value
	}
	return "data"
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	addr := addrFromEnv()
	dataDir := dataDirFromEnv()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-only-secret-change-me"
		logger.Warn("JWT_SECRET is not set, using the dev default")
	}

	st, err := store.Load(dataDir)
	if err != nil {
		logger.Error("load store", "err", err)
		os.Exit(1)
	}
	logger.Info("store loaded", "products", len(st.Rows.Products))

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewServer(st, logger, auth.NewManager(secret)).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
