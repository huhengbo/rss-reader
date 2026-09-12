package feed

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	appstate "rss-reader/internal/state"
)

func TestUpdateFeedChangesAfterPinnedItemAndKeepsFailureCache(t *testing.T) {
	var revision atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if revision.Load() == 2 {
			http.Error(w, "private upstream error", 503)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<rss version="2.0"><channel><title>Example</title><link>https://example.com</link><description>Fixture</description><item><guid>pinned</guid><title>Pinned</title><link>https://example.com/1</link></item><item><guid>second</guid><title>Revision %d</title><link>https://example.com/2</link></item></channel></rss>`, revision.Load())
	}))
	defer server.Close()
	s := appstate.New(config.Config{Values: []string{server.URL}})
	a, err := archive.Open(filepath.Join(t.TempDir(), "archive.txt"))
	if err != nil {
		t.Fatal(err)
	}
	UpdateFeed(context.Background(), s, a, server.URL, "ignored")
	revision.Store(1)
	UpdateFeed(context.Background(), s, a, server.URL, "ignored")
	if got := s.Snapshot().Sources[0].Items[1].Title; got != "Revision 1" {
		t.Fatalf("stale later item: %s", got)
	}
	revision.Store(2)
	UpdateFeed(context.Background(), s, a, server.URL, "ignored")
	if got := s.Snapshot().Sources[0]; got.Status != "stale" || len(got.Items) != 2 {
		t.Fatal("cache missing on fetch failure")
	}
}
