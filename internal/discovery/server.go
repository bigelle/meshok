package discovery

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

const (
	SERVICE_NAME string = "_meshok._tcp"
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
	info    ServerInfo
	entries chan *mdns.ServiceEntry
	ann     *mdns.Server
	logger  *slog.Logger
}

func (s *Server) ServerInfo() ServerInfo {
	return s.info
}

func (s *Server) Start(ctx context.Context) error {
	service, err := mdns.NewMDNSService(
		s.info.Instance,
		SERVICE_NAME,
		DOMAIN,
		s.info.Host+".",
		PORT,
		nil, // IPs
		nil, // TXT
	)
	if err != nil {
		return err
	}
	if s.logger == nil {
		panic("logger is nil")
	}

	return s.serve(service, ctx)
}

func (s *Server) serve(service *mdns.MDNSService, ctx context.Context) error {
	ann, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return err
	}
	s.ann = ann

	ticker := time.NewTicker(3 * time.Second)

	params := mdns.DefaultParams(service.Service)

	// this includes advertisements sent by this client
	rawEntries := make(chan *mdns.ServiceEntry, 4)
	params.Entries = rawEntries
	// and this is not
	s.entries = make(chan *mdns.ServiceEntry, 4)

	go func() {
		defer close(s.entries)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return

			case <-ticker.C:
				go func() {
					err = mdns.QueryContext(ctx, params)
					if err != nil {
						s.logger.Error("failed to query mDNS entries",
							"error", err,
						)
					}
				}()

			case entry, ok := <-rawEntries:
				if !ok {
					return
				}
				// filtering ourself out
				if isLocalEntry(entry, service) {
					continue
				}
				s.entries <- entry
			}
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	err := s.ann.Shutdown()
	return err
}

func (s *Server) Entries() (chan *mdns.ServiceEntry, error) {
	if s.entries == nil {
		return nil, errors.New("discovery server is shut down")
	}
	return s.entries, nil
}

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
