package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	appstate "rss-reader/internal/state"
)

func TestVersionedSnapshotAndLegacyEndpoint(t *testing.T) {
	s := appstate.New(config.Config{Values: []string{"https://example.com/rss?auth=secret-a", "https://example.com/rss?auth=secret-b"}})
	s.SetFeed("https://example.com/rss?auth=secret-a", domain.Feed{Title: "A", Link: "https://example.com"})
	app, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()
	response, err := http.Get(srv.URL + "/api/v1/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	var snap domain.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Version != 1 || len(snap.Sources) != 2 || snap.Sources[1].Status != "loading" {
		t.Fatalf("incomplete snapshot: %s", data)
	}
	if strings.Contains(string(data), "secret-") {
		t.Fatal("configured credentials exposed")
	}
	legacy, err := http.Get(srv.URL + "/feeds")
	if err != nil {
		t.Fatal(err)
	}
	defer legacy.Body.Close()
	var feeds []domain.Feed
	if err := json.NewDecoder(legacy.Body).Decode(&feeds); err != nil || len(feeds) != 1 {
		t.Fatal("legacy /feeds contract changed")
	}
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?v=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&snap); err != nil || len(snap.Sources) != 2 {
		t.Fatal("first frame is not an atomic snapshot")
	}
	_, _, err = conn.ReadMessage()
	if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		t.Fatalf("one-shot stream did not close normally: %v", err)
	}
}

func TestBootstrapEscapesUntrustedTextAndUnknownPaths(t *testing.T) {
	payload := "</script><script>alert(1)</script>"
	s := appstate.New(config.Config{WebTitle: payload})
	app, err := New(s)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 200 {
		t.Fatalf("home failed: %s", w.Body)
	}
	if strings.Contains(w.Body.String(), payload) {
		t.Fatal("bootstrap permits script breakout")
	}
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/not-found", nil))
	if w.Code != 404 {
		t.Fatal("unknown path should not render dashboard")
	}
}

func TestCloseTerminatesUpgradedConnections(t *testing.T) {
	app, err := New(appstate.New(config.Config{AutoUpdatePush: 1}))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?v=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var snap domain.Snapshot
	if err := conn.ReadJSON(&snap); err != nil {
		t.Fatal(err)
	}
	app.Close()
	app.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("upgraded connection survived application close")
	}
}
