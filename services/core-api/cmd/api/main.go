package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/httpapi"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := &http.Server{Addr: addr, Handler: httpapi.New(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("core API starting", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("core API stopped", "error", err)
		os.Exit(1)
	}
}
