package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nicumicle/parallel/internal/api"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error running server: %s", err.Error())
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	port := 8080
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		BaseContext:  func(l net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
		Handler:      api.NewAPI(),
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("Server started at: %d\n", port)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		stop()
	}

	return srv.Shutdown(context.Background())
}
