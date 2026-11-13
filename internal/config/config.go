package config

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bigelle/meshok/internal/database"
	"github.com/bigelle/meshok/internal/identity"
)

type Config struct {
	DeviceId string
}

type ConfigLoader struct {
	store  ConfigStore
	logger *slog.Logger
}

func NewConfigLoader(params LoaderParams) (*ConfigLoader, error) {
	if params.Store == nil {
		return nil, errors.New("config store is nil")
	}
	if params.Logger == nil {
		params.Logger = slog.Default()
	}

	return &ConfigLoader{
		store:  params.Store,
		logger: params.Logger,
	}, nil
}

// oh for fucks sake at least it's not ConfigLoaderConfig
type LoaderParams struct {
	Store  ConfigStore
	Logger *slog.Logger
}

type ConfigStore interface {
	LoadAllSettings(context.Context) ([]database.Setting, error)
	StoreSetting(context.Context, database.Setting) error
}

func (l *ConfigLoader) LoadConfig(ctx context.Context) (*Config, error) {
	rctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	settings, err := l.store.LoadAllSettings(rctx)
	if err != nil {
		return nil, err
	}

	var cfg Config
	// 1. if value is required but no fallback - err
	// 2. if value is required but there's a fallback - apply and store it
	// 3. if value is not required - skip
	for _, setting := range settings {
		switch setting.Key {
		case "device_id":
			if setting.Value == "" {
				setting.Value = identity.GenerateRandomDeviceId()
				wctx, wcancel := context.WithTimeout(ctx, 1*time.Second)
				defer wcancel()

				if err := l.store.StoreSetting(wctx, setting); err != nil {
					return nil, err
				}
			}
			cfg.DeviceId = setting.Value
		default:
			l.logger.Warn("unknown config key", "key", setting.Key)
		}
	}

	return &cfg, nil
}
