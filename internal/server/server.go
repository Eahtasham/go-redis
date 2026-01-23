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

const AOFPath = "appendonly.aof"

type Server struct {
	Listener *netlayer.Listener
	Store    *store.Store
	AOF      *persistence.AOF
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

	// Initialize AOF persistence
	aof, err := persistence.NewAOF(AOFPath)
	if err != nil {
		log.Printf("Warning: Could not initialize AOF: %v", err)
		// Continue without persistence
	}

	// Wire the store to handlers
	handlers.InitStore(s)

	// Wire AOF to handlers (may be nil if init failed)
	handlers.InitAOF(aof)

	// Register all command handlers
	handlers.RegisterAll()

	return &Server{
		Listener: ln,
		Store:    s,
		AOF:      aof,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Server) Start() error {
	fmt.Println("Server Starting on :6379")

	// Start AOF background writer
	if s.AOF != nil {
		s.AOF.Run()
		fmt.Println("AOF persistence enabled")
	}

	// Replay AOF to restore state (before accepting connections)
	count := 0
	persistence.Replay(AOFPath, func(v resp.Value) {
		commands.Dispatch(v)
		count++
	})
	if count > 0 {
		fmt.Printf("Replayed %d commands from AOF\n", count)
	}

	fmt.Println("Ready to accept connections")
	return s.Listener.Serve(s.ctx, netlayer.HandleConn)
}

func (s *Server) Shutdown() {
	fmt.Println("Shutting down server...")

	// Stop accepting new connections
	s.cancel()
	s.Listener.Close()

	// Stop AOF and ensure all pending writes are flushed
	if s.AOF != nil {
		s.AOF.Stop()
		fmt.Println("AOF flushed and closed")
	}

	fmt.Println("Server stopped")
}
