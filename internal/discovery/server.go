package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bigelle/meshok/internal/db"
	"github.com/brutella/dnssd"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/samber/lo"
)

const (
	SERVICE_TYPE string = "_meshok._tcp"
	PORT         int    = 52867
)

type ServerConfig struct {
	DeviceID  string // currently no use
	NetworkID string // currently no use
	Logger    *slog.Logger
}

type ServerInfo struct {
	Host     string
	Instance string
	DeviceID string
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("can not resolve system's host name")
	}
	return &Server{
		info: ServerInfo{
			Host:     host,
			Instance: NewInstanceName(),
			DeviceID: gonanoid.Must(),
		},
		logger: cfg.Logger,
	}, nil
}

type Server struct {
	info      ServerInfo
	logger    *slog.Logger
	handler   dnssd.ServiceHandle
	responder dnssd.Responder
	cancel    context.CancelFunc
	db        *db.DeviceDb
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

	sctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	return s.serve(sctx, sv)
}

func (s *Server) serve(ctx context.Context, sv dnssd.Service) error {
	rp, err := dnssd.NewResponder()
	if err != nil {
		return err
	}

	hdl, err := rp.Add(sv)
	if err != nil {
		return err
	}
	s.handler = hdl

	add := func(e dnssd.BrowseEntry) {
		id := e.Text["id"]
		// skipping
		if id == s.info.DeviceID {
			return
		}

		ips := mapIpsToString(e.IPs)

		wCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err := s.db.UpsertDevice(wCtx, db.Device{
			Id:         id,
			PseudoName: e.Name,
			IPs:        ips,
		})
		if err != nil {
			s.logger.Error("failed to create or update device", "error", err)
		}
	}

	rm := func(e dnssd.BrowseEntry) {
		id := e.Text["id"]
		// skipping
		if id == s.info.DeviceID {
			return
		}

		ips := mapIpsToString(e.IPs)

		wCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err := s.db.RemoveDeviceAddresses(wCtx, id, ips...)
		if err != nil {
			s.logger.Error("failed to remove device address", "error", err)
		}
	}

	go func() {
		if err := rp.Respond(ctx); err != nil {
			s.logger.Error("error while announcing service", "error", err)
		}
	}()
	go func() {
		svcName := fmt.Sprintf("%s.%s.", sv.Type, sv.Domain)
		if err := dnssd.LookupType(ctx, svcName, add, rm); err != nil {
			s.logger.Error("error while looking up the mDNS service", "service type", sv.Type, "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.cancel()
	s.responder.Remove(s.handler)
	return nil
}

func mapIpsToString(ips []net.IP) []string {
	raw := lo.Map(ips, func(x net.IP, _ int) string {
		return x.String()
	})
	return lo.Filter(raw, func(x string, _ int) bool {
		// empty addresses
		if x == "<nil>" {
			return false
		}
		// malformed addresses
		if strings.HasPrefix(x, "?") {
			return false
		}
		return true
	})
}
