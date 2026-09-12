package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	"rss-reader/internal/feed"
	appstate "rss-reader/internal/state"
	"rss-reader/internal/web"
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

	go feed.UpdateFeeds(state, archiveStore)
	go feed.WatchConfigFileChanges("config.json", state, archiveStore)

	upgrader := &websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/feeds", func(w http.ResponseWriter, r *http.Request) {
		getFeedsHandler(state, w, r)
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(state, upgrader, w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tplHandler(state, w, r)
	})

	fs := http.FileServer(http.FS(web.Static))
	mux.Handle("/static/", fs)

	serve := fmt.Sprintf(":%d", conf.Port)
	log.Fatal(http.ListenAndServe(serve, mux))
}

func tplHandler(state *appstate.State, w http.ResponseWriter, r *http.Request) {
	tmplInstance := template.New("index.html").Delims("<<", ">>")
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}
	tmpl, err := tmplInstance.Funcs(funcMap).ParseFS(web.Static, "static/index.html")
	if err != nil {
		log.Println("模板加载错误:", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	conf := state.Config()
	data := struct {
		Keywords       string
		RssDataList    []domain.Feed
		AutoUpdatePush int
		ListHeight     int
		WebTitle       string
		WebDes         string
	}{
		Keywords:       getKeywords(state),
		RssDataList:    feed.GetFeeds(state),
		AutoUpdatePush: conf.AutoUpdatePush,
		ListHeight:     conf.ListHeight,
		WebTitle:       conf.WebTitle,
		WebDes:         conf.WebDes,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("模板渲染错误:", err)
	}
}

func wsHandler(state *appstate.State, upgrader *websocket.Upgrader, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		conf := state.Config()
		for _, url := range conf.Values {
			cache, ok := state.Feed(url)
			if !ok {
				log.Printf("Error getting feed from db is null %v", url)
				continue
			}
			data, err := json.Marshal(cache)
			if err != nil {
				log.Printf("json marshal failure: %s", err.Error())
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Error sending message or Connection closed: %v", err)
				return
			}
		}

		if conf.AutoUpdatePush == 0 {
			return
		}
		time.Sleep(time.Duration(conf.AutoUpdatePush) * time.Minute)
	}
}

func getKeywords(state *appstate.State) string {
	words := ""
	conf := state.Config()
	for _, url := range conf.Values {
		cache, ok := state.Feed(url)
		if !ok {
			continue
		}
		if cache.Title != "" {
			words += cache.Title + ","
		}
	}
	return words
}

func getFeedsHandler(state *appstate.State, w http.ResponseWriter, r *http.Request) {
	feeds := feed.GetFeeds(state)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		log.Printf("encode feeds response: %v", err)
	}
}
