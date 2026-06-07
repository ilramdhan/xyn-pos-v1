package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	inventory "github.com/xyn-pos/services/inventory"
)

func main() {
	cfg, err := inventory.ConfigFromEnv()
	if err != nil {
		slog.Error("inventory: config error", "err", err)
		os.Exit(1)
	}

	app, err := inventory.NewServer(cfg)
	if err != nil {
		slog.Error("inventory: server init error", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("inventory: service starting", "grpc_port", cfg.GRPCPort)
	if err := app.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("inventory: service error", "err", err)
	}
}
