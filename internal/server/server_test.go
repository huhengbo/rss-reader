package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	appstate "rss-reader/internal/state"
)

func TestHandlerServesFeeds(t *testing.T) {
	state := appstate.New(config.Config{
		Values: []string{"https://example.com/feed.xml"},
	})
	state.SetFeed("https://example.com/feed.xml", domain.Feed{
		Title: "Example Feed",
		Link:  "https://example.com",
	})

	srv, err := New(state)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/feeds", nil)
	response := httptest.NewRecorder()
	srv.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	var feeds []domain.Feed
	if err := json.NewDecoder(response.Body).Decode(&feeds); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(feeds) != 1 || feeds[0].Title != "Example Feed" {
		t.Fatalf("feeds = %#v, want one Example Feed", feeds)
	}
}

func TestHandlerRendersHomePage(t *testing.T) {
	state := appstate.New(config.Config{
		WebTitle:   "Test RSS Reader",
		WebDes:     "Test description",
		ListHeight: 600,
	})

	srv, err := New(state)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	srv.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "Test RSS Reader") {
		t.Fatal("rendered page does not contain configured title")
	}
}
