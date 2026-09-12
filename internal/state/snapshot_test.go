package state

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"rss-reader/internal/config"
	"rss-reader/internal/domain"
)

func TestSnapshotIdentityOrderAndPrivacy(t *testing.T) {
	a, b := "https://user:secret@example.com/rss?auth=private", "https://example.com/rss?category=b"
	s := New(config.Config{Values: []string{a, b}})
	s.SetFeed(a, domain.Feed{Title: "A", Link: "https://example.com"})
	s.SetFeed(b, domain.Feed{Title: "B", Link: "https://example.com"})
	first := s.Snapshot()
	if len(first.Sources) != 2 || first.Sources[0].ID == first.Sources[1].ID {
		t.Fatal("sources were merged by homepage")
	}
	encoded, _ := json.Marshal(first)
	for _, secret := range []string{"private", "user:secret", "auth="} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("snapshot exposes configured credentials")
		}
	}
	s.ReplaceConfig(config.Config{Values: []string{b, a}})
	if s.Snapshot().Sources[1].ID != first.Sources[0].ID {
		t.Fatal("source ID depends on order")
	}
	s.ReplaceConfig(config.Config{})
	if len(s.Snapshot().Sources) != 0 {
		t.Fatal("deleted sources remain in snapshot")
	}
}

func TestFetchStateAndContentTimestamps(t *testing.T) {
	const url = "https://example.com/rss"
	s := New(config.Config{Values: []string{url}})
	if s.Snapshot().Sources[0].Status != "loading" {
		t.Fatal("missing source should be visible as loading")
	}
	at := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	f := domain.Feed{Title: "A", Items: []domain.Item{{Title: "pinned", Link: "https://example.com/1"}}}
	ticket, ok := s.BeginFetch(url, at)
	if !ok {
		t.Fatal("fetch rejected")
	}
	s.CompleteFetch(url, ticket, f, at, nil)
	first := s.Snapshot().Sources[0]
	ticket, _ = s.BeginFetch(url, at.Add(time.Minute))
	s.CompleteFetch(url, ticket, f, at.Add(time.Minute), nil)
	second := s.Snapshot().Sources[0]
	if first.LastSuccessAt == second.LastSuccessAt || first.ContentChangedAt != second.ContentChangedAt {
		t.Fatal("success time and content change time are conflated")
	}
	f.Items = append(f.Items, domain.Item{Title: "new", Link: "https://example.com/2"})
	ticket, _ = s.BeginFetch(url, at.Add(2*time.Minute))
	s.CompleteFetch(url, ticket, f, at.Add(2*time.Minute), nil)
	if len(s.Snapshot().Sources[0].Items) != 2 {
		t.Fatal("later item lost behind pinned first item")
	}
	ticket, _ = s.BeginFetch(url, at.Add(3*time.Minute))
	s.CompleteFetch(url, ticket, domain.Feed{}, at.Add(3*time.Minute), errors.New("auth=secret"))
	stale := s.Snapshot().Sources[0]
	if stale.Status != "stale" || len(stale.Items) != 2 || strings.Contains(stale.Message, "secret") {
		t.Fatal("failure should preserve cache and redact errors")
	}
}

func TestRemovedInFlightFetchCannotResurrectSource(t *testing.T) {
	const url = "https://example.com/rss"
	s := New(config.Config{Values: []string{url}})
	ticket, _ := s.BeginFetch(url, time.Now())
	if _, ok := s.BeginFetch(url, time.Now()); ok {
		t.Fatal("overlapping fetch accepted")
	}
	s.ReplaceConfig(config.Config{})
	s.ReplaceConfig(config.Config{Values: []string{url}})
	if s.CompleteFetch(url, ticket, domain.Feed{Title: "obsolete"}, time.Now(), nil) {
		t.Fatal("obsolete fetch accepted after removal/re-add")
	}
}

func TestSnapshotIsDetached(t *testing.T) {
	s := New(config.Config{Values: []string{"a"}})
	s.SetFeed("a", domain.Feed{Title: "A", Items: []domain.Item{{Title: "original"}}})
	x := s.Snapshot()
	x.Sources[0].Items[0].Title = "mutated"
	if s.Snapshot().Sources[0].Items[0].Title != "original" {
		t.Fatal("snapshot aliases state")
	}
}
