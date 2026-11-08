package db_test

import (
	"log/slog"
	"slices"
	"testing"

	database "github.com/bigelle/meshok/internal/db"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestConnectionsDb(t *testing.T) {
	g, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)
	err = g.AutoMigrate(&database.Device{})
	require.NoError(t, err)

	db := database.NewDeviceDb(g, slog.New(slog.DiscardHandler))
	require.NotNil(t, db)

	d := database.Device{
		Id: "SomeAsciiId",
		IPs: datatypes.JSONSlice[string]{
			"192.168.1.1",
			"fe80::200:5aee:feaa:20a2",
		},
		PseudoName: "SomePseudoName",
	}
	t.Run("create device", func(t *testing.T) {
		err = db.UpsertDevice(t.Context(), d)

		require.NoError(t, err)
	})

	if t.Failed() {
		t.Skip("failed to create device, skipping")
	}

	t.Run("get device", func(t *testing.T) {
		res, err := db.GetDeviceById(t.Context(), "SomeAsciiId")
		require.NoError(t, err)

		assert.Equal(t, &d, res)
	})

	if t.Failed() {
		t.Skip("GetDeviceById is not working, skipping")
	}

	t.Run("remove address", func(t *testing.T) {
		err = db.RemoveDeviceAddresses(t.Context(), "SomeAsciiId", "192.168.1.1")
		require.NoError(t, err)

		res, err := db.GetDeviceById(t.Context(), "SomeAsciiId")
		require.NoError(t, err)

		assert.Equal(t, datatypes.JSONSlice[string]{"fe80::200:5aee:feaa:20a2"}, res.IPs)
	})

	t.Run("upsert device with multiple new addresses", func(t *testing.T) {
		d.IPs = append(d.IPs, "10.10.0.100", "192.168.2.3")
		err = db.UpsertDevice(t.Context(), d)
		require.NoError(t, err)

		res, err := db.GetDeviceById(t.Context(), "SomeAsciiId")
		require.NoError(t, err)

		expect := datatypes.JSONSlice[string]{"fe80::200:5aee:feaa:20a2", "192.168.1.1", "10.10.0.100", "192.168.2.3"}
		got := res.IPs
		slices.Sort(expect)
		slices.Sort(got)

		assert.Equal(t, expect, got)
	})
}
