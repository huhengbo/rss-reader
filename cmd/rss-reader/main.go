package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	"rss-reader/internal/feed"
	"rss-reader/internal/server"
	appstate "rss-reader/internal/state"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	conf, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	state := appstate.New(conf)
	archiveStore, err := archive.Open(conf.Archives)
	if err != nil {
		return fmt.Errorf("open archive store: %w", err)
	}

	appServer, err := server.New(state)
	if err != nil {
		return fmt.Errorf("create HTTP server: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath := config.Path()
	go feed.UpdateFeeds(ctx, state, archiveStore)
	go feed.WatchConfigFileChanges(ctx, configPath, state, archiveStore)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", conf.Port),
		Handler:           appServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case err := <-errCh:
		stop()
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}
