package discovery

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/bigelle/meshok/internal/database"
	"github.com/brutella/dnssd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type fakeStore struct {
	upserts  []database.Device
	removals []fakeRemoval
}

type fakeRemoval struct {
	id    string
	addrs []string
}

func (f *fakeStore) UpsertDevice(_ context.Context, d database.Device) error {
	f.upserts = append(f.upserts, d)
	return nil
}

func (f *fakeStore) RemoveDeviceAddresses(_ context.Context, id string, addr ...string) error {
	f.removals = append(f.removals, fakeRemoval{id: id, addrs: addr})
	return nil
}

func newTestServer(deviceID string, store DeviceStore) *Server {
	return &Server{
		info: ServerInfo{
			Host:     "test-host",
			Instance: "TestInstance",
			DeviceID: deviceID,
		},
		logger: slog.New(slog.DiscardHandler),
		store:  store,
	}
}

func TestNewServerValidation(t *testing.T) {
	t.Run("empty device id", func(t *testing.T) {
		_, err := NewServer(Config{DeviceStore: &fakeStore{}})
		require.Error(t, err)
	})

	t.Run("nil device store", func(t *testing.T) {
		_, err := NewServer(Config{DeviceID: "device-1"})
		require.Error(t, err)
	})
}

func TestNewServer(t *testing.T) {
	s, err := NewServer(Config{
		DeviceID:    "device-1",
		DeviceStore: &fakeStore{},
	})
	require.NoError(t, err)

	host, err := os.Hostname()
	require.NoError(t, err)

	info := s.ServerInfo()
	assert.Equal(t, host, info.Host)
	assert.Equal(t, "device-1", info.DeviceID)
	assert.NotEmpty(t, info.Instance)
}

func TestMapIPsToString(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("fe80::200:5aee:feaa:20a2"),
	}

	assert.Equal(t, []string{"192.168.1.1", "fe80::200:5aee:feaa:20a2"}, mapIpsToString(ips))
}

func TestServerAdd(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer("self", store)

	add := s.add(context.Background())

	add(dnssd.BrowseEntry{
		Name: "HappyOtter",
		IPs: []net.IP{
			net.ParseIP("192.168.1.10"),
			net.ParseIP("fe80::1"),
		},
		Text: map[string]string{"id": "peer-1"},
	})

	require.Len(t, store.upserts, 1)
	assert.Equal(t, database.Device{
		Id:           "peer-1",
		InstanceName: "HappyOtter",
		IPs:          datatypes.JSONSlice[string]{"192.168.1.10", "fe80::1"},
	}, store.upserts[0])
}

func TestServerAddIgnoresSelfAndEntriesWithoutID(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer("self", store)

	add := s.add(context.Background())

	// entry ID is the client's own ID
	add(dnssd.BrowseEntry{
		Name: "Self",
		IPs:  []net.IP{net.ParseIP("192.168.1.11")},
		Text: map[string]string{"id": "self"},
	})
	// no ID (no TXT)
	add(dnssd.BrowseEntry{
		Name: "ProbeWithoutTxt",
		IPs:  []net.IP{net.ParseIP("192.168.1.12")},
		Text: map[string]string{},
	})

	assert.Empty(t, store.upserts)
}

func TestServerRemove(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer("self", store)

	remove := s.remove(context.Background())

	remove(dnssd.BrowseEntry{
		IPs: []net.IP{
			net.ParseIP("192.168.1.10"),
			net.ParseIP("fe80::1"),
		},
		Text: map[string]string{"id": "peer-1"},
	})

	require.Len(t, store.removals, 1)
	assert.Equal(t, "peer-1", store.removals[0].id)
	assert.Equal(t, []string{"192.168.1.10", "fe80::1"}, store.removals[0].addrs)
}

func TestServerRemoveIgnoresSelfAndEntriesWithoutID(t *testing.T) {
	store := &fakeStore{}
	s := newTestServer("self", store)

	remove := s.remove(context.Background())

	// entry ID is the client's own ID
	remove(dnssd.BrowseEntry{
		IPs:  []net.IP{net.ParseIP("192.168.1.11")},
		Text: map[string]string{"id": "self"},
	})
	// no ID (no TXT)
	remove(dnssd.BrowseEntry{
		IPs:  []net.IP{net.ParseIP("192.168.1.12")},
		Text: map[string]string{},
	})

	assert.Empty(t, store.removals)
}
