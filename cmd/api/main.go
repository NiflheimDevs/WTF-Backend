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

	"gitlab.chabokan.net/niflheim/wtf-backend/internal/app"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/config"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/handler"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	container, err := app.NewContainer(ctx, cfg)
	if err != nil {
		logger.Error("application wiring failed", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	router := handler.NewRouter(handler.Dependencies{
		Health:  container.DB,
		Auth:    container.Auth,
		Regions: container.Regions,
		Request: container.Requests,
		Metrics: container.Metrics,
	}, handler.RouterConfig{
		AllowedOrigin:     cfg.FrontendOrigin,
		RequestsPerMinute: cfg.RequestsPerMinute,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
	}
}
