package logger_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/bigelle/meshok/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func withStateHome(t *testing.T, dir string) {
	t.Helper()

	oldStateHome := xdg.StateHome
	xdg.StateHome = dir
	t.Cleanup(func() { xdg.StateHome = oldStateHome })
}

func TestNew(t *testing.T) {
	withStateHome(t, t.TempDir())

	log, closeFn, err := logger.New(zap.NewAtomicLevelAt(zap.InfoLevel))
	require.NoError(t, err)
	require.NotNil(t, log)
	require.NotNil(t, closeFn)

	log.Info("hello from test", "key", "value")
	closeFn()

	data, err := os.ReadFile(filepath.Join(xdg.StateHome, "meshok", "meshok.log"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello from test")
}

func TestNewMissingStateHome(t *testing.T) {
	withStateHome(t, "")

	log, closeFn, err := logger.New(zap.NewAtomicLevelAt(zap.InfoLevel))
	require.Error(t, err)
	assert.Nil(t, log)
	assert.Nil(t, closeFn)
}
