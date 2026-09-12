package main

import (
	"fmt"
	"log"
	"net/http"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	"rss-reader/internal/feed"
	"rss-reader/internal/server"
	appstate "rss-reader/internal/state"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	state := appstate.New(conf)
	archiveStore, err := archive.Open(conf.Archives)
	if err != nil {
		log.Fatalf("open archive store: %v", err)
	}

	httpServer, err := server.New(state)
	if err != nil {
		log.Fatalf("create HTTP server: %v", err)
	}

	configPath := config.Path()
	go feed.UpdateFeeds(state, archiveStore)
	go feed.WatchConfigFileChanges(configPath, state, archiveStore)

	address := fmt.Sprintf(":%d", conf.Port)
	log.Fatal(http.ListenAndServe(address, httpServer.Handler()))
}
