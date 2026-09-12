package utils

import (
	"testing"

	"github.com/mmcdole/gofeed"

	"rss-reader/globals"
	"rss-reader/models"
)

func TestGetFeedsPreservesConfiguredOrderAndSkipsMissing(t *testing.T) {
	originalConfig := globals.RssUrls
	originalDB := globals.DbMap
	defer func() {
		globals.RssUrls = originalConfig
		globals.DbMap = originalDB
	}()

	globals.RssUrls = models.Config{
		Values: []string{
			"https://example.com/first.xml",
			"https://example.com/missing.xml",
			"https://example.com/second.xml",
		},
	}
	globals.DbMap = map[string]models.Feed{
		"https://example.com/first.xml":  {Title: "First"},
		"https://example.com/second.xml": {Title: "Second"},
	}

	feeds := GetFeeds()
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
		{
			name:  "trims surrounding whitespace",
			input: "  https://example.com/post  ",
			want:  "https://example.com/post",
		},
		{
			name:  "removes query parameters",
			input: "https://example.com/post?utm_source=rss&id=1",
			want:  "https://example.com/post",
		},
		{
			name:  "removes fragment",
			input: "https://example.com/post#comments",
			want:  "https://example.com/post",
		},
		{
			name:  "removes query and fragment",
			input: "https://example.com/post?id=1#comments",
			want:  "https://example.com/post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLink(tt.input); got != tt.want {
				t.Fatalf("normalizeLink(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckHandlesEmptyFeedState(t *testing.T) {
	originalConfig := globals.RssUrls
	originalDB := globals.DbMap
	originalHash := globals.Hash
	originalMatchList := globals.MatchList
	defer func() {
		globals.RssUrls = originalConfig
		globals.DbMap = originalDB
		globals.Hash = originalHash
		globals.MatchList = originalMatchList
	}()

	const feedURL = "https://example.com/feed.xml"
	globals.RssUrls = models.Config{}
	globals.DbMap = map[string]models.Feed{
		feedURL: {Title: "Cached feed", Items: nil},
	}
	globals.Hash = map[string]int{}
	globals.MatchList = []string{"never-match-this-title"}

	item := &gofeed.Item{
		Title: "Example item",
		Link:  "https://example.com/post?id=1",
	}
	result := &gofeed.Feed{Items: []*gofeed.Item{item}}

	Check(feedURL, result, item)
	Check(feedURL, &gofeed.Feed{}, item)
	Check(feedURL, nil, item)
	Check(feedURL, result, nil)
}
