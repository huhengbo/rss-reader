package utils

import (
	"testing"

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
