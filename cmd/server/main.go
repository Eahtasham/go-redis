package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/Eahtasham/go-redis/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := server.New(":6379")

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	srv.Shutdown()
}
