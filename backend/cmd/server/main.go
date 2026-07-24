package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/demo/smart-home/backend/internal/adapter/hass"
	"github.com/demo/smart-home/backend/internal/config"
	"github.com/demo/smart-home/backend/internal/httpserver"
	"github.com/demo/smart-home/backend/internal/infra"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	ctx := context.Background()

	db, err := infra.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := infra.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	defer func() { _ = rdb.Close() }()

	ha := hass.NewClient(cfg.HassBaseURL, cfg.HassToken, cfg.HassTimeout)

	srv := httpserver.New(cfg, log, db, rdb, ha)
	srv.StartHub(ctx)
	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("smart-home-service listening",
			"addr", cfg.HTTPAddr(),
			"env", cfg.AppEnv,
			"hass_configured", ha.Configured(),
		)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "err", err)
	}
	log.Info("stopped")
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}
