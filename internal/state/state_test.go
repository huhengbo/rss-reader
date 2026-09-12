package state

import (
	"testing"

	"rss-reader/models"
)

func TestStateConfigReturnsSnapshot(t *testing.T) {
	state := New(models.Config{
		Values:   []string{"https://example.com/a.xml"},
		Keywords: []string{"alpha"},
	})

	config := state.Config()
	config.Values[0] = "https://example.com/changed.xml"
	config.Keywords[0] = "changed"

	fresh := state.Config()
	if fresh.Values[0] != "https://example.com/a.xml" {
		t.Fatalf("config values mutated through snapshot: %#v", fresh.Values)
	}
	if fresh.Keywords[0] != "alpha" {
		t.Fatalf("config keywords mutated through snapshot: %#v", fresh.Keywords)
	}
}

func TestStateFeedsFollowConfiguredOrder(t *testing.T) {
	state := New(models.Config{
		Values: []string{
			"https://example.com/second.xml",
			"https://example.com/missing.xml",
			"https://example.com/first.xml",
		},
	})
	state.SetFeed("https://example.com/first.xml", models.Feed{Title: "First"})
	state.SetFeed("https://example.com/second.xml", models.Feed{Title: "Second"})

	feeds := state.Feeds()
	if len(feeds) != 2 {
		t.Fatalf("Feeds() length = %d, want 2", len(feeds))
	}
	if feeds[0].Title != "Second" || feeds[1].Title != "First" {
		t.Fatalf("Feeds() titles = [%q, %q], want [Second, First]", feeds[0].Title, feeds[1].Title)
	}
}

func TestStateReplaceConfig(t *testing.T) {
	state := New(models.Config{Port: 8080})
	state.ReplaceConfig(models.Config{Port: 9090, Keywords: []string{"new"}})

	config := state.Config()
	if config.Port != 9090 {
		t.Fatalf("Config().Port = %d, want 9090", config.Port)
	}
	if len(config.Keywords) != 1 || config.Keywords[0] != "new" {
		t.Fatalf("Config().Keywords = %#v, want [new]", config.Keywords)
	}
}
