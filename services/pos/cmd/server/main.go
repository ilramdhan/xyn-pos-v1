package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	pos "github.com/xyn-pos/services/pos"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg, err := pos.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	app, err := pos.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize pos service", "err", err)
		os.Exit(1)
	}
	defer app.Stop()

	slog.Info("pos service starting", "port", cfg.GRPCPort)
	if err := app.Start(ctx); err != nil {
		slog.Error("pos service stopped", "err", err)
	}
}
