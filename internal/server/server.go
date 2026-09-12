package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	state       *appstate.State
	template    *template.Template
	upgrader    websocket.Upgrader
	mu          sync.Mutex
	closed      bool
	done        chan struct{}
	connections map[*websocket.Conn]struct{}
}

func New(state *appstate.State) (*Server, error) {
	funcs := template.FuncMap{"statusText": func(status string) string {
		switch status {
		case "ready":
			return "已同步"
		case "empty":
			return "暂无文章"
		case "error":
			return "获取失败"
		case "stale":
			return "旧缓存"
		default:
			return "加载中"
		}
	}}
	tmpl, err := template.New("index.html").Delims("<<", ">>").Funcs(funcs).ParseFS(web.Static, "static/index.html")
	if err != nil {
		return nil, fmt.Errorf("parse web template: %w", err)
	}
	return &Server{state: state, template: tmpl, upgrader: websocket.Upgrader{CheckOrigin: sameOrigin}, done: make(chan struct{}), connections: make(map[*websocket.Conn]struct{})}, nil
}

// Close terminates upgraded connections, which http.Server.Shutdown does not own.
func (s *Server) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.done)
	connections := make([]*websocket.Conn, 0, len(s.connections))
	for conn := range s.connections {
		connections = append(connections, conn)
	}
	s.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthHandler)
	mux.HandleFunc("GET /feeds", s.getFeedsHandler)
	mux.HandleFunc("GET /api/v1/snapshot", s.snapshotHandler)
	mux.HandleFunc("GET /ws", s.wsHandler)
	mux.HandleFunc("GET /{$}", s.tplHandler)
	mux.Handle("GET /static/", http.FileServer(http.FS(web.Static)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// Assets are small and versioned with the binary; no immutable caching on mutable URLs.
		w.Header().Set("Cache-Control", "no-cache")
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) tplHandler(w http.ResponseWriter, _ *http.Request) {
	snapshot := s.state.Snapshot()
	if snapshot.Title == "" {
		snapshot.Title = "RSS Reader"
	}
	if snapshot.ListHeight < 180 {
		snapshot.ListHeight = 600
	}
	data := struct {
		Snapshot    domain.Snapshot
		BlankSource domain.Source
		BlankItem   domain.Item
	}{Snapshot: snapshot}
	var buf bytes.Buffer
	// html/template encodes the struct as escaped JSON inside application/json.
	if err := s.template.Execute(&buf, data); err != nil {
		log.Printf("render template: %v", err)
		http.Error(w, "render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) snapshotHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(s.state.Snapshot()); err != nil {
		log.Printf("encode snapshot: %v", err)
	}
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("v")
	if version != "" && version != "1" {
		http.Error(w, "unsupported snapshot protocol", http.StatusBadRequest)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = conn.Close()
		return
	}
	s.connections[conn] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.connections, conn); s.mu.Unlock(); _ = conn.Close() }()
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(websocketPongWait)) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	revision := ""
	write := func() error {
		if version == "" {
			return s.writeFeeds(conn)
		}
		snapshot := s.state.Snapshot()
		if snapshot.Revision == revision {
			return nil
		}
		if err := conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
			return err
		}
		if err := conn.WriteJSON(snapshot); err != nil {
			return err
		}
		revision = snapshot.Revision
		return nil
	}
	if err := write(); err != nil {
		return
	}
	closeNormally := func() {
		if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "snapshot complete"), time.Now().Add(websocketWriteTimeout)); err != nil {
			return
		}
		// Let the peer acknowledge Close before tearing down TCP. An immediate
		// Close can race Firefox's opening handshake and discard the snapshot.
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case <-done:
		case <-s.done:
		case <-r.Context().Done():
		case <-timer.C:
		}
	}
	if s.state.Config().AutoUpdatePush == 0 {
		closeNormally()
		return
	}
	period := func() time.Duration {
		if version == "1" {
			return time.Second
		}
		minutes := s.state.Config().AutoUpdatePush
		if minutes < 1 {
			minutes = 1
		}
		return time.Duration(minutes) * time.Minute
	}
	update := time.NewTimer(period())
	defer update.Stop()
	ping := time.NewTicker(websocketPingPeriod)
	defer ping.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-r.Context().Done():
			return
		case <-done:
			return
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(websocketWriteTimeout)); err != nil {
				return
			}
		case <-update.C:
			if err := write(); err != nil {
				return
			}
			if s.state.Config().AutoUpdatePush == 0 {
				closeNormally()
				return
			}
			update.Reset(period())
		}
	}
}

// writeFeeds preserves the pre-v1 WebSocket wire format for existing clients.
func (s *Server) writeFeeds(conn *websocket.Conn) error {
	for _, feed := range s.state.Feeds() {
		if err := conn.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
			return err
		}
		if err := conn.WriteJSON(feed); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getFeedsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(s.state.Feeds()); err != nil {
		log.Printf("encode feeds: %v", err)
	}
}

func sameOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && strings.EqualFold(parsed.Host, r.Host)
}
