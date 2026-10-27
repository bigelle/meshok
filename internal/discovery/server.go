package discovery

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/hashicorp/mdns"
	"go.uber.org/zap"
)

const (
	SERVICE_NAME string = "_meshok._tcp"
	DOMAIN       string = "local."
	PORT         int    = 52867
)

type ServerConfig struct {
	DeviceID  string // currently no use
	NetworkID string // currently no use
	Logger    *zap.Logger
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("can not resolve system's host name")
	}
	host += "."
	return &Server{
		host:     host,
		instance: NewInstanceName(),
		logger:   cfg.Logger,
	}, nil
}

type Server struct {
	host     string
	instance string
	entries  chan *mdns.ServiceEntry
	ann      *mdns.Server
	logger   *zap.Logger
}

func (s *Server) Start(ctx context.Context) error {
	service, err := mdns.NewMDNSService(
		s.instance,
		SERVICE_NAME,
		DOMAIN,
		s.host,
		PORT,
		nil, // IPs
		nil, // TXT
	)
	if err != nil {
		return err
	}
	if s.logger == nil {
		// FIXME: idk in which case it could happen,
		// but if this ever returns an error,
		// i should handle it
		s.logger, _ = zap.NewDevelopment()
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
						s.logger.Error("failed to query mDNS entries", zap.Error(err))
					}
				}()

			case entry, ok := <-rawEntries:
				if !ok {
					return
				}
				// filtering ourself out
				// FIXME: this is not enough (but for now it works fine)
				if entry.Name == service.Instance+"._meshok._tcp.local." {
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
