package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/bigelle/meshok/internal/database"
	"github.com/brutella/dnssd"
)

const (
	SERVICE_TYPE string = "_meshok._tcp"
	PORT         int    = 52867
)

type Config struct {
	DeviceID  string // currently no use
	NetworkID string // currently no use

	Logger      *slog.Logger
	DeviceStore DeviceStore
}

type ServerInfo struct {
	Host     string
	Instance string
	DeviceID string
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.DeviceID == "" {
		return nil, errors.New("device id is empty")
	}
	// TODO: check for network id if i ever use it

	if cfg.DeviceStore == nil {
		return nil, errors.New("device storage is nil")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to retriev OS hostname: %w", err)
	}

	return &Server{
		info: ServerInfo{
			Host:     host,
			Instance: NewInstanceName(),
			DeviceID: cfg.DeviceID,
		},
		logger: cfg.Logger,
		store:  cfg.DeviceStore,
	}, nil
}

type Server struct {
	info      ServerInfo
	logger    *slog.Logger
	handler   dnssd.ServiceHandle
	responder dnssd.Responder
	store     DeviceStore
}

func (s *Server) ServerInfo() ServerInfo {
	return s.info
}

func (s *Server) Start(ctx context.Context) error {
	cfg := dnssd.Config{
		Name: s.info.Instance,
		Type: SERVICE_TYPE,
		Host: s.info.Host,
		Port: PORT,
		Text: map[string]string{
			"id": s.info.DeviceID,
		},
	}

	sv, err := dnssd.NewService(cfg)
	if err != nil {
		return err
	}

	// probing for instance name availability for a maximum of 15 seconds
	// and auto-incrementing the name if there's a conflict
	pctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	sv, err = dnssd.ProbeService(pctx, sv)
	cancel()
	if err != nil {
		return err
	}
	// in case ProbeService incremeneted instance name
	s.info.Instance = sv.Name

	return s.serve(ctx, sv)
}

func (s *Server) serve(ctx context.Context, sv dnssd.Service) error {
	rp, err := dnssd.NewResponder()
	if err != nil {
		return err
	}
	s.responder = rp

	hdl, err := rp.Add(sv)
	if err != nil {
		return err
	}
	s.handler = hdl

	go func() {
		if err := rp.Respond(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Error("error while announcing service", "error", err)
		}
	}()
	go func() {
		svcName := fmt.Sprintf("%s.%s.", sv.Type, sv.Domain)
		if err := dnssd.LookupType(ctx, svcName, s.add(ctx), s.remove(ctx)); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Error("error while looking up the mDNS service", "service type", sv.Type, "error", err)
		}
	}()

	return nil
}

func (s *Server) add(ctx context.Context) func(dnssd.BrowseEntry) {
	return func(be dnssd.BrowseEntry) {
		id := be.Text["id"]
		// the first sighting is a probe query without TXT records
		if id == "" {
			return
		}
		// skipping if entry came from this client
		if id == s.info.DeviceID {
			return
		}

		ips := mapIpsToString(be.IPs)

		wCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err := s.store.UpsertDevice(wCtx, database.Device{
			Id:           id,
			InstanceName: be.Name,
			IPs:          ips,
		})
		if err != nil {
			s.logger.Error("failed to create or update device", "error", err)
		}
	}
}

func (s *Server) remove(ctx context.Context) func(dnssd.BrowseEntry) {
	return func(be dnssd.BrowseEntry) {
		id := be.Text["id"]
		// an entry without an ID was never stored
		if id == "" {
			return
		}
		// skipping
		if id == s.info.DeviceID {
			return
		}

		ips := mapIpsToString(be.IPs)

		wCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err := s.store.RemoveDeviceAddresses(wCtx, id, ips...)
		if err != nil {
			s.logger.Error("failed to remove IP address", "error", err)
		}
	}
}

func (s *Server) Stop() {
	s.logger.Info("stopping discovery server")
	s.responder.Remove(s.handler)
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	s.Stop()
	return nil
}

func mapIpsToString(ips []net.IP) []string {
	var strs []string
	for _, ip := range ips {
		if ip.To16() != nil {
			strs = append(strs, ip.String())
		}
	}
	return strs
}

type DeviceStore interface {
	UpsertDevice(ctx context.Context, d database.Device) error
	RemoveDeviceAddresses(ctx context.Context, id string, addr ...string) error
}
