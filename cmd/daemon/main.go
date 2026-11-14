package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bigelle/meshok/internal/config"
	"github.com/bigelle/meshok/internal/database"
	"github.com/bigelle/meshok/internal/discovery"
	"github.com/bigelle/meshok/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	log, closeLog := setupLogger()
	defer closeLog()

	ctx, stop := setupSignals(log)
	defer stop()

	db := setupDb(log)

	cfg := loadConfig(ctx, db, log)

	d := setupDiscovery(db, cfg, log)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return d.Run(ctx) })

	if err := group.Wait(); err != nil {
		log.Error("error from discovery server", "error", err)
		os.Exit(1)
	}
}

func setupLogger() (log *slog.Logger, closure func()) {
	// FIXME: read log level from env or flags
	log, closure, err := logger.New(zap.NewAtomicLevelAt(zap.DebugLevel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logger: %s", err.Error())
		os.Exit(1)
	}
	return log, closure
}

func setupSignals(log *slog.Logger) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	log.Debug("listening for OS signals",
		"signals", []string{"Interrupt", "SIGTERM"},
	)
	return ctx, stop
}

func setupDb(log *slog.Logger) *gorm.DB {
	db, err := database.SetupDB()
	if err != nil {
		log.Error("failed to initiate database connection", "error", err)
		os.Exit(1)
	}
	return db
}

func setupDiscovery(db *gorm.DB, cfg *config.Config, log *slog.Logger) *discovery.Server {
	deviceStore := database.NewDeviceStore(db)
	d, err := discovery.NewServer(discovery.Config{
		DeviceID:    cfg.DeviceId,
		Logger:      log,
		DeviceStore: &deviceStore,
	})
	if err != nil {
		log.Error("failed to create discovery service", "error", err)
		os.Exit(1)
	}

	info := d.ServerInfo()
	log.Debug("created mDNS server",
		"hostname", info.Host,
		"instance name", info.Instance,
	)

	return d
}

func loadConfig(ctx context.Context, db *gorm.DB, log *slog.Logger) *config.Config {
	store := database.NewSettingsStore(db)
	loader, err := config.NewConfigLoader(config.LoaderParams{
		Logger: log,
		Store:  store,
	})
	if err != nil {
		log.Error("failed to initiate config loader", "error", err)
		os.Exit(1)
	}

	rctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	cfg, err := loader.LoadConfig(rctx)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	return cfg
}
