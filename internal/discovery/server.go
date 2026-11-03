package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/brutella/dnssd"
	"github.com/hashicorp/mdns"
)

const (
	SERVICE_TYPE string = "_meshok._tcp"
	DOMAIN       string = "local."
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
	}

	sv, err := dnssd.NewService(cfg)
	if err != nil {
		return err
	}

	// probing for instance name availability for at maximum of 15 seconds
	// and auto-incrementing the name if there's a conflict.
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
		// TODO: save entry into SQLite if it's:
		// 1. not from loopback
		// 2. new
		// or add a new address if the entry is from an already known host
		fmt.Printf("new entry: %+v\n", e)
	}

	rm := func(e dnssd.BrowseEntry) {
		// TODO: remove an entry from SQLite if:
		// 1. it's in the DB
		// 2. removed address is the last known address for this host
		// or just only remove the address if there's more
		fmt.Printf("removed entry: %+v\n", e)
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

// NOTE: do i still need it?
func isLocalEntry(newEntry *mdns.ServiceEntry, thisService *mdns.MDNSService) bool {
	// filter out loopbacks
	if newEntry.AddrV4.IsLoopback() {
		return true
	}
	if newEntry.AddrV6.IsLoopback() {
		return true
	}

	// filter out same host
	if newEntry.Host == thisService.HostName {
		return true
	}
	return false
}
