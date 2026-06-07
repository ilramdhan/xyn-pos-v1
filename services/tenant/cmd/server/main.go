package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tenant "github.com/xyn-pos/services/tenant"
)

func main() {
	ctx := context.Background()

	cfg, err := tenant.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	app, err := tenant.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize service", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("tenant-service starting", "port", cfg.GRPCPort, "env", cfg.Env)

	go func() {
		if err := app.Start(); err != nil {
			slog.Error("gRPC server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down tenant-service...")
	app.Stop()
}
