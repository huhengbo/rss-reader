package feed

import (
	"path/filepath"
	"testing"

	"github.com/mmcdole/gofeed"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	appstate "rss-reader/internal/state"
)

func TestGetFeedsPreservesConfiguredOrderAndSkipsMissing(t *testing.T) {
	state := appstate.New(config.Config{
		Values: []string{
			"https://example.com/first.xml",
			"https://example.com/missing.xml",
			"https://example.com/second.xml",
		},
	})
	state.SetFeed("https://example.com/first.xml", domain.Feed{Title: "First"})
	state.SetFeed("https://example.com/second.xml", domain.Feed{Title: "Second"})

	feeds := GetFeeds(state)
	if len(feeds) != 2 {
		t.Fatalf("GetFeeds() returned %d feeds, want 2", len(feeds))
	}
	if feeds[0].Title != "First" || feeds[1].Title != "Second" {
		t.Fatalf("GetFeeds() titles = [%q, %q], want [First, Second]", feeds[0].Title, feeds[1].Title)
	}
}

func TestNormalizeLink(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trims surrounding whitespace", input: "  https://example.com/post  ", want: "https://example.com/post"},
		{name: "removes query parameters", input: "https://example.com/post?utm_source=rss&id=1", want: "https://example.com/post"},
		{name: "removes fragment", input: "https://example.com/post#comments", want: "https://example.com/post"},
		{name: "removes query and fragment", input: "https://example.com/post?id=1#comments", want: "https://example.com/post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLink(tt.input); got != tt.want {
				t.Fatalf("normalizeLink(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeURLForLogRedactsCredentialsAndQuery(t *testing.T) {
	got := safeURLForLog("https://user:pass@example.com/feed.xml?auth=secret#section")
	want := "https://example.com/feed.xml"
	if got != want {
		t.Fatalf("safeURLForLog() = %q, want %q", got, want)
	}
}

func TestCheckHandlesEmptyFeedState(t *testing.T) {
	const feedURL = "https://example.com/feed.xml"
	state := appstate.New(config.Config{Keywords: []string{"never-match-this-title"}})
	state.SetFeed(feedURL, domain.Feed{Title: "Cached feed", Items: nil})

	archiveStore, err := archive.Open(filepath.Join(t.TempDir(), "archives.txt"))
	if err != nil {
		t.Fatalf("archive.Open() error = %v", err)
	}

	item := &gofeed.Item{Title: "Example item", Link: "https://example.com/post?id=1"}
	result := &gofeed.Feed{Items: []*gofeed.Item{item}}

	Check(state, archiveStore, feedURL, result, item)
	Check(state, archiveStore, feedURL, &gofeed.Feed{}, item)
	Check(state, archiveStore, feedURL, nil, item)
	Check(state, archiveStore, feedURL, result, nil)
}
