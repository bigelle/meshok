package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigelle/meshok/internal/discovery"
	"github.com/bigelle/meshok/internal/logger"
	"go.uber.org/zap"
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

	d, err := discovery.NewServer(&discovery.ServerConfig{
		Logger: logger,
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

	if err := d.Start(ctx); err != nil {
		logger.Error("failed to start discovery server", "error", err)
		os.Exit(1)
	}
	logger.Debug("mDNS server is online",
		"advertising port", discovery.PORT,
	)

	entries, _ := d.Entries()
	for e := range entries {
		fmt.Printf("new entry: %+v\n", e)
	}
}
