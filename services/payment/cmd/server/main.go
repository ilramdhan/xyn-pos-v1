package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	payment "github.com/xyn-pos/services/payment"
)

func main() {
	cfg, err := payment.ConfigFromEnv()
	if err != nil {
		slog.Error("payment: config error", "err", err)
		os.Exit(1)
	}

	app, err := payment.NewServer(cfg)
	if err != nil {
		slog.Error("payment: server init error", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("payment: service starting", "grpc_port", cfg.GRPCPort, "http_port", cfg.HTTPPort)
	if err := app.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("payment: service error", "err", err)
	}
}
