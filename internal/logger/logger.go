package logger

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(level zap.AtomicLevel) (log *slog.Logger, closure func(), err error) {
	dir := filepath.Join(xdg.StateHome, "meshok")
	if dir == "meshok" {
		return nil, nil, errors.New("log directory is not defined") // could phrase it better
	}

	// Chmod 700 (chmod a+rwx,g-rwx,o-rwx) sets permissions so that,
	// (U)ser / owner can read, can write and can execute.
	// (G)roup can't read, can't write and can't execute.
	// (O)thers can't read, can't write and can't execute.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, nil, err
	}

	logfilePath := filepath.Join(dir, "meshok.log")
	logfile := &lumberjack.Logger{
		Filename:   logfilePath,
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 5,
		Compress:   true,
	}

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logfile),
		level,
	)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stderr),
		level,
	)

	core := zapcore.NewTee(fileCore, consoleCore)
	zl := zap.New(core)

	log = slog.New(zapslog.NewHandler(core))
	slog.SetDefault(log)
	return log, func() { zl.Sync(); logfile.Close() }, nil
}
