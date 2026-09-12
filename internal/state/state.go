package state

import (
	"sync"

	"rss-reader/internal/config"
	"rss-reader/internal/domain"
)

// State owns the mutable in-memory application state.
// Callers receive snapshots instead of direct access to shared maps/configuration.
type State struct {
	mu     sync.RWMutex
	config config.Config
	feeds  map[string]domain.Feed
}

func New(conf config.Config) *State {
	return &State{
		config: cloneConfig(conf),
		feeds:  make(map[string]domain.Feed),
	}
}

func (s *State) Config() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

func (s *State) ReplaceConfig(conf config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cloneConfig(conf)
}

func (s *State) Feed(url string) (domain.Feed, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	feed, ok := s.feeds[url]
	return feed, ok
}

func (s *State) SetFeed(url string, feed domain.Feed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.feeds[url] = feed
}

func (s *State) Feeds() []domain.Feed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feeds := make([]domain.Feed, 0, len(s.config.Values))
	for _, url := range s.config.Values {
		feed, ok := s.feeds[url]
		if !ok {
			continue
		}
		feeds = append(feeds, feed)
	}
	return feeds
}

func cloneConfig(conf config.Config) config.Config {
	cloned := conf
	cloned.Values = append([]string(nil), conf.Values...)
	cloned.Keywords = append([]string(nil), conf.Keywords...)
	return cloned
}
