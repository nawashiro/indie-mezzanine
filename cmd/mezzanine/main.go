package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mezzanine/internal/relay"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		addr := os.Getenv("HEALTH_URL")
		if addr == "" {
			addr = "http://127.0.0.1:8080/healthz"
		}
		c := http.Client{Timeout: 2 * time.Second}
		r, e := c.Get(addr)
		if e != nil {
			return e
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return fmt.Errorf("health HTTP %d", r.StatusCode)
		}
		return nil
	}
	c, e := relay.ConfigFromEnv()
	if e != nil {
		return e
	}
	s, e := relay.OpenStore(c.DataDir)
	if e != nil {
		return e
	}
	defer s.Close()
	f, e := relay.NewSafeFetcher(c)
	if e != nil {
		return e
	}
	app, e := relay.NewApp(c, s, f)
	if e != nil {
		return e
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e = app.Start(ctx); e != nil {
		return e
	}
	server := app.Server()
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	log.Printf("mezzanine listening on %s", c.Listen)
	select {
	case e = <-result:
		cancel()
	case <-ctx.Done():
	}
	shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	shutdownErr := server.Shutdown(shutdownCtx)
	cancel()
	app.Wait()
	if e != nil && !errors.Is(e, http.ErrServerClosed) {
		return e
	}
	return shutdownErr
}
