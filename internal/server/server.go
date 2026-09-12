package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"rss-reader/internal/domain"
	appstate "rss-reader/internal/state"
	"rss-reader/internal/web"
)

type Server struct {
	state    *appstate.State
	template *template.Template
	upgrader websocket.Upgrader
}

func New(state *appstate.State) (*Server, error) {
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}
	tmpl, err := template.New("index.html").Delims("<<", ">>").Funcs(funcMap).ParseFS(web.Static, "static/index.html")
	if err != nil {
		return nil, fmt.Errorf("parse web template: %w", err)
	}

	return &Server{
		state:    state,
		template: tmpl,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/feeds", s.getFeedsHandler)
	mux.HandleFunc("/ws", s.wsHandler)
	mux.HandleFunc("/", s.tplHandler)
	mux.Handle("/static/", http.FileServer(http.FS(web.Static)))
	return mux
}

func (s *Server) tplHandler(w http.ResponseWriter, r *http.Request) {
	conf := s.state.Config()
	data := struct {
		Keywords       string
		RssDataList    []domain.Feed
		AutoUpdatePush int
		ListHeight     int
		WebTitle       string
		WebDes         string
	}{
		Keywords:       s.getKeywords(),
		RssDataList:    s.state.Feeds(),
		AutoUpdatePush: conf.AutoUpdatePush,
		ListHeight:     conf.ListHeight,
		WebTitle:       conf.WebTitle,
		WebDes:         conf.WebDes,
	}

	if err := s.template.Execute(w, data); err != nil {
		log.Printf("render template: %v", err)
	}
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade websocket: %v", err)
		return
	}
	defer conn.Close()

	for {
		conf := s.state.Config()
		for _, url := range conf.Values {
			cached, ok := s.state.Feed(url)
			if !ok {
				continue
			}
			data, err := json.Marshal(cached)
			if err != nil {
				log.Printf("marshal websocket feed: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("write websocket message: %v", err)
				return
			}
		}

		if conf.AutoUpdatePush == 0 {
			return
		}
		time.Sleep(time.Duration(conf.AutoUpdatePush) * time.Minute)
	}
}

func (s *Server) getKeywords() string {
	words := ""
	conf := s.state.Config()
	for _, url := range conf.Values {
		cached, ok := s.state.Feed(url)
		if !ok || cached.Title == "" {
			continue
		}
		words += cached.Title + ","
	}
	return words
}

func (s *Server) getFeedsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.state.Feeds()); err != nil {
		log.Printf("encode feeds response: %v", err)
	}
}
