package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/bigelle/meshok/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupDB(t *testing.T) {
	dir := t.TempDir()

	g, err := database.SetupDB(dir)
	require.NoError(t, err)

	assert.True(t, g.Migrator().HasTable(&database.Device{}))

	_, err = os.Stat(filepath.Join(dir, "meshok.db"))
	require.NoError(t, err)
}

func TestSetupDBDefaultPath(t *testing.T) {
	oldStateHome := xdg.StateHome
	xdg.StateHome = t.TempDir()
	t.Cleanup(func() { xdg.StateHome = oldStateHome })

	g, err := database.SetupDB()
	require.NoError(t, err)

	assert.True(t, g.Migrator().HasTable(&database.Device{}))

	_, err = os.Stat(filepath.Join(xdg.StateHome, "meshok", "meshok.db"))
	require.NoError(t, err)
}
