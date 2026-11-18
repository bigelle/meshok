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
	"github.com/bigelle/meshok/internal/state"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func main() {
	cfg := loadConfig()

	log, closeLog := setupLogger(cfg)
	defer closeLog()

	ctx, stop := setupSignals(log)
	defer stop()

	db := setupDb(log)

	state := loadAppState(ctx, db, log)

	d := setupDiscovery(db, state.DeviceId, log)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return d.Run(ctx) })

	if err := group.Wait(); err != nil {
		log.Error("error from discovery server", "error", err)
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	cfg, err := config.LoadConfig()
	if err != nil {
		// FIXME: It shouldn't be as important
		// so I need to throw a warning and fallback to defaults instead
		// but for now I can't just remove the os.Exit()
		// so I leave it there
		fmt.Fprintf(os.Stderr, "failed to load config: %s", err.Error())
		os.Exit(1)
	}
	return cfg
}

func setupLogger(cfg *config.Config) (log *slog.Logger, closure func()) {
	log, closure, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logger: %s", err.Error())
		os.Exit(1)
	}
	return log, closure
}

func setupSignals(log *slog.Logger) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func setupDb(log *slog.Logger) *gorm.DB {
	db, err := database.SetupDB()
	if err != nil {
		log.Error("failed to initiate database connection", "error", err)
		os.Exit(1)
	}
	log.Info("successfully opened sqlite database")
	return db
}

func loadAppState(ctx context.Context, db *gorm.DB, log *slog.Logger) *state.State {
	store := database.NewSettingsStore(db)
	loader, err := state.NewStateLoader(state.LoaderParams{
		Logger: log,
		Store:  store,
	})
	if err != nil {
		log.Error("failed to initiate config loader", "error", err)
		os.Exit(1)
	}

	rctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	cfg, err := loader.LoadState(rctx)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	log.Info("successfully loaded app state from sqlite")
	return cfg
}

func setupDiscovery(db *gorm.DB, deviceId string, log *slog.Logger) *discovery.Server {
	deviceStore := database.NewDeviceStore(db)
	d, err := discovery.NewServer(discovery.Config{
		DeviceID:    deviceId,
		Logger:      log,
		DeviceStore: &deviceStore,
	})
	if err != nil {
		log.Error("failed to create discovery service", "error", err)
		os.Exit(1)
	}

	info := d.ServerInfo()
	log.Info("started mDNS service",
		"instance name", info.Instance,
		"advertising port", discovery.PORT,
	)

	return d
}
