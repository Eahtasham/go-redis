package server

import (
	"context"

	"github.com/Eahtasham/go-redis/internal/netlayer"
)

type Server struct {
	Listener *netlayer.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(addr string) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	ln, _ := netlayer.Newlistener(addr)

	return &Server{
		Listener: ln,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Server) Start() error {
	return s.Listener.Serve(s.ctx, netlayer.HandleConn)
}

func (s *Server) Shutdown() {
	s.cancel()
	s.Listener.Close()
}
