package database_test

import (
	"log/slog"
	"slices"
	"testing"

	database1 "github.com/bigelle/meshok/internal/database"
	database "github.com/bigelle/meshok/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestConnectionsDb(t *testing.T) {
	g, err := database1.SetupDB(t.TempDir())
	require.NoError(t, err)
	err = g.AutoMigrate(&database.Device{})
	require.NoError(t, err)

	db := database.NewDeviceStore(g, slog.New(slog.DiscardHandler))
	require.NotNil(t, db)

	d := database.Device{
		Id: "SomeAsciiId",
		IPs: datatypes.JSONSlice[string]{
			"192.168.1.1",
			"fe80::200:5aee:feaa:20a2",
		},
		InstanceName: "SomePseudoName",
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

	t.Run("upsert device with multiple new addresses and a new instance name", func(t *testing.T) {
		d.IPs = append(d.IPs, "10.10.0.100", "192.168.2.3")
		d.InstanceName = "SomeNewInstanceName"
		err = db.UpsertDevice(t.Context(), d)
		require.NoError(t, err)

		res, err := db.GetDeviceById(t.Context(), "SomeAsciiId")
		require.NoError(t, err)

		assert.Equal(t, d.InstanceName, res.InstanceName)

		expect := datatypes.JSONSlice[string]{"fe80::200:5aee:feaa:20a2", "192.168.1.1", "10.10.0.100", "192.168.2.3"}
		got := res.IPs
		slices.Sort(expect)
		slices.Sort(got)

		assert.Equal(t, expect, got)
	})
}
