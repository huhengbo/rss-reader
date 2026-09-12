package state

import (
	"sync"

	"rss-reader/models"
)

// State owns the mutable in-memory application state.
// Callers receive snapshots instead of direct access to shared maps/configuration.
type State struct {
	mu     sync.RWMutex
	config models.Config
	feeds  map[string]models.Feed
}

func New(config models.Config) *State {
	return &State{
		config: cloneConfig(config),
		feeds:  make(map[string]models.Feed),
	}
}

func (s *State) Config() models.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

func (s *State) ReplaceConfig(config models.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cloneConfig(config)
}

func (s *State) Feed(url string) (models.Feed, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	feed, ok := s.feeds[url]
	return feed, ok
}

func (s *State) SetFeed(url string, feed models.Feed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feeds[url] = feed
}

func (s *State) Feeds() []models.Feed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feeds := make([]models.Feed, 0, len(s.config.Values))
	for _, url := range s.config.Values {
		feed, ok := s.feeds[url]
		if !ok {
			continue
		}
		feeds = append(feeds, feed)
	}
	return feeds
}

func cloneConfig(config models.Config) models.Config {
	cloned := config
	cloned.Values = append([]string(nil), config.Values...)
	cloned.Keywords = append([]string(nil), config.Keywords...)
	return cloned
}
