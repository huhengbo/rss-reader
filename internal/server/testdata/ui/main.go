// This executable is test-only (under testdata); it is not part of the shipped command.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	"rss-reader/internal/server"
	appstate "rss-reader/internal/state"
)

var urls = []string{"https://example.com/rss?source=a&auth=fixture-private", "https://example.com/rss?source=b", "https://example.com/rss?source=empty"}

func main() {
	state := appstate.New(config.Config{})
	reset := func() {
		state.ReplaceConfig(config.Config{})
		state.ReplaceConfig(config.Config{Values: urls, WebTitle: "我的订阅", WebDes: "按自己的节奏，阅读感兴趣的内容。", AutoUpdatePush: 1, ListHeight: 600})
		for index, title := range []string{"技术周刊", "同站专题", "空订阅"} {
			feed := domain.Feed{Title: title, Link: "https://example.com", Items: []domain.Item{}}
			if index < 2 { for i := 0; i < 12; i++ { feed.Items = append(feed.Items, domain.Item{GUID: fmt.Sprintf("item-%d", i), Title: fmt.Sprintf("%s：第 %d 篇阅读与开发笔记", title, i+1), Link: fmt.Sprintf("https://example.com/article?id=%d", i+index*100), PublishedAt: "2026-09-12T00:00:00Z"}) } }
			at := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
			ticket, _ := state.BeginFetch(urls[index], at)
			state.CompleteFetch(urls[index], ticket, feed, at, nil)
		}
	}
	reset()
	app, err := server.New(state); if err != nil { log.Fatal(err) }; defer app.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /fixture/{action}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("action") {
		case "reset": reset()
		case "new":
			f, _ := state.Feed(urls[0]); f.Items = append(f.Items, domain.Item{GUID: "fresh", Title: "新文章：可靠的实时阅读", Link: "https://example.com/fresh"}); state.SetFeed(urls[0], f)
		case "delete": c := state.Config(); c.Values = urls[1:]; state.ReplaceConfig(c)
		case "empty": c := state.Config(); c.Values = nil; state.ReplaceConfig(c)
		case "zero": c := state.Config(); c.AutoUpdatePush = 0; state.ReplaceConfig(c)
		case "fail": ticket, _ := state.BeginFetch(urls[0], time.Now()); state.CompleteFetch(urls[0], ticket, domain.Feed{}, time.Now(), errors.New("do not disclose this upstream URL"))
		case "long": f, _ := state.Feed(urls[0]); f.Title = strings.Repeat("长标题", 20); f.Items[0].Title = strings.Repeat("LongUnbrokenTitle", 30); state.SetFeed(urls[0], f)
		default: http.NotFound(w, r); return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", app.Handler())
	// Never bind fixture mutation routes to an external interface.
	log.Fatal(http.ListenAndServe("127.0.0.1:4173", mux))
}
