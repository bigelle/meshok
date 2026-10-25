package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigelle/meshok/internal/discovery"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		// in which case this could happen?
		log.Fatal("failed to start zap logger")
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d := discovery.NewServer()

	if err := d.Start(ctx); err != nil {
		logger.Fatal("failed to start discovery server", zap.Error(err))
	}

	entries, _ := d.Entries()
	for e := range entries {
		fmt.Printf("new entry: %+v\n", e)
	}
}
