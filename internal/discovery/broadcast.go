package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/mdns"
)

func NewServer() *Server {
	return &Server{}
}

type Server struct {
	ch  chan *mdns.ServiceEntry
	ann *mdns.Server
}

func (s *Server) Start(ctx context.Context) error {
	service, err := mdns.NewMDNSService("host", "_meshok._tcp", "", "", 52867, nil, nil)
	if err != nil {
		return err
	}

	ch := make(chan *mdns.ServiceEntry, 4)
	s.ch = ch

	return s.serve(service, ctx)
}

func (s *Server) serve(service *mdns.MDNSService, ctx context.Context) error {
	ann, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return err
	}
	s.ann = ann

	mdns.Lookup(service.Service, s.ch)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			// Do i need to do something with entries here?
			// case entry := <- ch:
			default:
				continue
			}
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	err := s.ann.Shutdown()
	close(s.ch)
	return err
}

func (s *Server) Entries() (chan *mdns.ServiceEntry, error) {
	if s.ch == nil {
		return nil, errors.New("discovery server is shut down")
	}
	return s.ch, nil
}
