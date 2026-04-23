package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rodrigo/home-monitor/internal/api"
	"github.com/rodrigo/home-monitor/internal/config"
	"github.com/rodrigo/home-monitor/internal/storage"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.Load(logger)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		logger.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("database ready", "path", cfg.DBPath)

	queries := storage.New(db)

	srv := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler: api.NewServer(logger, queries, cfg.APIKey),
	}

	logger.Info("home-monitor API starting", "addr", srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}
