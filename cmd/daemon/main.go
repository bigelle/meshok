package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigelle/meshok/internal/database"
	"github.com/bigelle/meshok/internal/discovery"
	"github.com/bigelle/meshok/internal/logger"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	logger, closure, err := logger.New(zap.NewAtomicLevelAt(zap.DebugLevel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up logger: %s", err.Error())
		os.Exit(1)
	}
	defer closure()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Debug("listening for OS signals",
		"signals", []string{"Interrupt", "SIGTERM"},
	)

	db, err := database.SetupDB()
	if err != nil {
		logger.Error("failed to initiate database connection", "error", err)
		os.Exit(1)
	}
	deviceStore := database.NewDeviceStore(db)

	d, err := discovery.NewServer(discovery.Config{
		DeviceID:    gonanoid.Must(),
		Logger:      logger,
		DeviceStore: &deviceStore,
	})
	if err != nil {
		logger.Error("failed to create discovery service", "error", err)
		os.Exit(1)
	}

	info := d.ServerInfo()
	logger.Debug("created mDNS server",
		"hostname", info.Host,
		"instance name", info.Instance,
	)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return d.Run(ctx) })

	if err := group.Wait(); err != nil {
		logger.Error("error from discovery server", "error", err)
		os.Exit(1)
	}
}
