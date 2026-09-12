package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"rss-reader/globals"
	"rss-reader/internal/archive"
	appstate "rss-reader/internal/state"
	"rss-reader/models"
	"rss-reader/utils"
)

func main() {
	config, err := models.ParseConf()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	state := appstate.New(config)
	archiveStore, err := archive.Open(config.Archives)
	if err != nil {
		log.Fatalf("open archive store: %v", err)
	}

	go utils.UpdateFeeds(state, archiveStore)
	go utils.WatchConfigFileChanges("config.json", state, archiveStore)

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

	fs := http.FileServer(http.FS(globals.DirStatic))
	mux.Handle("/static/", fs)

	serve := fmt.Sprintf(":%d", config.Port)
	log.Fatal(http.ListenAndServe(serve, mux))
}

func tplHandler(state *appstate.State, w http.ResponseWriter, r *http.Request) {
	tmplInstance := template.New("index.html").Delims("<<", ">>")
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}
	tmpl, err := tmplInstance.Funcs(funcMap).ParseFS(globals.DirStatic, "static/index.html")
	if err != nil {
		log.Println("模板加载错误:", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	config := state.Config()
	data := struct {
		Keywords       string
		RssDataList    []models.Feed
		AutoUpdatePush int
		ListHeight     int
		WebTitle       string
		WebDes         string
	}{
		Keywords:       getKeywords(state),
		RssDataList:    utils.GetFeeds(state),
		AutoUpdatePush: config.AutoUpdatePush,
		ListHeight:     config.ListHeight,
		WebTitle:       config.WebTitle,
		WebDes:         config.WebDes,
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
		config := state.Config()
		for _, url := range config.Values {
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

		if config.AutoUpdatePush == 0 {
			return
		}
		time.Sleep(time.Duration(config.AutoUpdatePush) * time.Minute)
	}
}

func getKeywords(state *appstate.State) string {
	words := ""
	config := state.Config()
	for _, url := range config.Values {
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
	feeds := utils.GetFeeds(state)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		log.Printf("encode feeds response: %v", err)
	}
}
