package server

import (
	"context"
	"fmt"
	"log"

	"github.com/Eahtasham/go-redis/internal/commands"
	"github.com/Eahtasham/go-redis/internal/commands/handlers"
	"github.com/Eahtasham/go-redis/internal/engine/store"
	"github.com/Eahtasham/go-redis/internal/netlayer"
	"github.com/Eahtasham/go-redis/internal/persistence"
	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

type Server struct {
	Listener *netlayer.Listener
	Store    *store.Store
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(addr string) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	ln, err := netlayer.NewListener(addr)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the store
	s := store.NewStore()

	// Wire the store to handlers
	handlers.InitStore(s)

	// Register all command handlers
	handlers.RegisterAll()

	return &Server{
		Listener: ln,
		Store:    s,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Server) Start() error {
	fmt.Println("Server Starting on :6379")

	// Replay AOF to restore state
	persistence.Replay("appendonly.aof", func(v resp.Value) {
		commands.Dispatch(v)
	})

	fmt.Println("Ready to accept connections")
	return s.Listener.Serve(s.ctx, netlayer.HandleConn)
}

func (s *Server) Shutdown() {
	s.cancel()
	s.Listener.Close()
}
