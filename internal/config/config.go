package config

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	LogLevel zap.AtomicLevel
}

type config struct {
	// one of: debug, info, warn, error
	LogLevel string `mapstructure:"log_level"`
}

func LoadConfig(paths ...string) (*Config, error) {
	cfg, err := loadConfig(paths...)
	if err != nil {
		return nil, err
	}

	return marshalConfig(cfg)
}

func loadConfig(paths ...string) (*config, error) {
	v := viper.New()
	v.SetConfigName("meshok")
	v.SetConfigType("toml")

	v.SetDefault("log_level", "info")

	for _, path := range paths {
		v.AddConfigPath(path)
	}

	v.ReadInConfig()

	v.SetEnvPrefix("MESHOK")
	v.AutomaticEnv()

	var cfg config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func marshalConfig(raw *config) (*Config, error) {
	var cfg Config

	switch raw.LogLevel {
	case "debug":
		cfg.LogLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.LogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.LogLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.LogLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return nil, fmt.Errorf("unsupported log level: %s", raw.LogLevel)
	}

	return &cfg, nil
}
