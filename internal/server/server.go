package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"rss-reader/internal/domain"
	appstate "rss-reader/internal/state"
	"rss-reader/internal/web"
)

const (
	websocketWriteTimeout = 10 * time.Second
	websocketPongWait     = 60 * time.Second
	websocketPingPeriod   = 30 * time.Second
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
		upgrader: websocket.Upgrader{CheckOrigin: sameOrigin},
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/feeds", s.getFeedsHandler)
	mux.HandleFunc("/ws", s.wsHandler)
	mux.HandleFunc("/", s.tplHandler)
	mux.Handle("/static/", http.FileServer(http.FS(web.Static)))
	return mux
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
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

	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	if err := s.writeFeeds(conn); err != nil {
		log.Printf("write websocket feeds: %v", err)
		return
	}

	conf := s.state.Config()
	if conf.AutoUpdatePush == 0 {
		return
	}

	updateTimer := time.NewTimer(time.Duration(conf.AutoUpdatePush) * time.Minute)
	defer updateTimer.Stop()
	pingTicker := time.NewTicker(websocketPingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case <-pingTicker.C:
			deadline := time.Now().Add(websocketWriteTimeout)
			if err := conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return
			}
		case <-updateTimer.C:
			if err := s.writeFeeds(conn); err != nil {
				log.Printf("write websocket feeds: %v", err)
				return
			}
			conf = s.state.Config()
			if conf.AutoUpdatePush == 0 {
				return
			}
			updateTimer.Reset(time.Duration(conf.AutoUpdatePush) * time.Minute)
		}
	}
}

func (s *Server) writeFeeds(conn *websocket.Conn) error {
	conf := s.state.Config()
	for _, feedURL := range conf.Values {
		cached, ok := s.state.Feed(feedURL)
		if !ok {
			continue
		}
		data, err := json.Marshal(cached)
		if err != nil {
			return fmt.Errorf("marshal feed: %w", err)
		}
		if err := conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getKeywords() string {
	words := ""
	conf := s.state.Config()
	for _, feedURL := range conf.Values {
		cached, ok := s.state.Feed(feedURL)
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

func sameOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}
