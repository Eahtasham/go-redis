package server

import (
	"context"
	"fmt"
	"log"

	// "github.com/Eahtasham/go-redis/internal/commands"
	// "github.com/Eahtasham/go-redis/internal/protocol/resp"
	"github.com/Eahtasham/go-redis/internal/netlayer"
)

type Server struct {
	Listener *netlayer.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(addr string) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	ln, err := netlayer.NewListener(addr)

	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		Listener: ln,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Server) Start() error {
	fmt.Println("Server Starting")
	// commands.Register("PING", func(args []string) resp.Value {
	// 	return resp.Value{
	// 		Type: resp.SimpleString,
	// 		Str:  "PONG",
	// 	}
	// })

	return s.Listener.Serve(s.ctx, netlayer.HandleConn)
}

func (s *Server) Shutdown() {
	s.cancel()
	s.Listener.Close()
}
