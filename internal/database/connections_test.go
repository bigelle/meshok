package database_test

import (
	"slices"
	"testing"

	"github.com/bigelle/meshok/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newTestStore(t *testing.T) *database.DeviceDb {
	t.Helper()

	g, err := database.SetupDB(t.TempDir())
	require.NoError(t, err)

	db := database.NewDeviceStore(g)
	return &db
}

func TestCreateAndGetDevice(t *testing.T) {
	db := newTestStore(t)

	d := database.Device{
		Id:           "SomeAsciiId",
		InstanceName: "SomeInstanceName",
		PseudoName:   "SomePseudoName",
		IPs: datatypes.JSONSlice[string]{
			"192.168.1.1",
			"fe80::200:5aee:feaa:20a2",
		},
	}

	require.NoError(t, db.UpsertDevice(t.Context(), d))

	res, err := db.GetDeviceById(t.Context(), "SomeAsciiId")
	require.NoError(t, err)
	assert.Equal(t, &d, res)
}

func TestGetDeviceByIdNotFound(t *testing.T) {
	db := newTestStore(t)

	res, err := db.GetDeviceById(t.Context(), "missing")
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestUpsertDeviceMergesAddressesAndUpdatesInstanceName(t *testing.T) {
	db := newTestStore(t)

	original := database.Device{
		Id:           "device-1",
		InstanceName: "OldName",
		IPs:          datatypes.JSONSlice[string]{"192.168.1.1", "fe80::1"},
	}
	require.NoError(t, db.UpsertDevice(t.Context(), original))

	update := database.Device{
		Id:           "device-1",
		InstanceName: "NewName",
		// fe80::1 is already known and must not be duplicated
		IPs: datatypes.JSONSlice[string]{"10.10.0.1", "fe80::1"},
	}
	require.NoError(t, db.UpsertDevice(t.Context(), update))

	res, err := db.GetDeviceById(t.Context(), "device-1")
	require.NoError(t, err)

	assert.Equal(t, "NewName", res.InstanceName)

	expect := datatypes.JSONSlice[string]{"192.168.1.1", "fe80::1", "10.10.0.1"}
	slices.Sort(expect)
	got := res.IPs
	slices.Sort(got)
	assert.Equal(t, expect, got)
}

func TestUpsertDeviceKeepsInstanceNameWhenEmpty(t *testing.T) {
	db := newTestStore(t)

	original := database.Device{
		Id:           "device-1",
		InstanceName: "KeepMe",
		IPs:          datatypes.JSONSlice[string]{"192.168.1.1"},
	}
	require.NoError(t, db.UpsertDevice(t.Context(), original))

	update := database.Device{
		Id:  "device-1",
		IPs: datatypes.JSONSlice[string]{"192.168.1.2"},
	}
	require.NoError(t, db.UpsertDevice(t.Context(), update))

	res, err := db.GetDeviceById(t.Context(), "device-1")
	require.NoError(t, err)
	assert.Equal(t, "KeepMe", res.InstanceName)
}

func TestRemoveDeviceAddresses(t *testing.T) {
	db := newTestStore(t)

	d := database.Device{
		Id:  "device-1",
		IPs: datatypes.JSONSlice[string]{"192.168.1.1", "192.168.1.2", "fe80::1"},
	}
	require.NoError(t, db.UpsertDevice(t.Context(), d))

	require.NoError(t, db.RemoveDeviceAddresses(t.Context(), "device-1", "192.168.1.1", "fe80::1"))

	res, err := db.GetDeviceById(t.Context(), "device-1")
	require.NoError(t, err)
	assert.Equal(t, datatypes.JSONSlice[string]{"192.168.1.2"}, res.IPs)
}
