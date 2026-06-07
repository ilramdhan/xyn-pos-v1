package main

import (
	"log/slog"
	"os"

	payment "github.com/xyn-pos/services/payment"
)

func main() {
	cfg, err := payment.ConfigFromEnv()
	if err != nil {
		slog.Error("payment: config error", "err", err)
		os.Exit(1)
	}

	srv, err := payment.NewServer(cfg)
	if err != nil {
		slog.Error("payment: server init error", "err", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		slog.Error("payment: server error", "err", err)
		os.Exit(1)
	}
}
