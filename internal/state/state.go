package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"rss-reader/internal/config"
	"rss-reader/internal/domain"
)

// State owns the mutable application data. All public reads are detached copies.
type State struct {
	mu sync.RWMutex
	config config.Config
	feeds map[string]domain.Feed
	status map[string]sourceStatus
	inflight map[string]uint64
	nextTicket uint64
}

type sourceStatus struct {
	attempt, success, changed string
	failed bool
}

func New(conf config.Config) *State {
	return &State{config: cloneConfig(conf), feeds: make(map[string]domain.Feed), status: make(map[string]sourceStatus), inflight: make(map[string]uint64)}
}

func (s *State) Config() config.Config {
	s.mu.RLock(); defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

func (s *State) ReplaceConfig(conf config.Config) {
	s.mu.Lock(); defer s.mu.Unlock()
	s.config = cloneConfig(conf)
	for raw := range s.feeds { if !s.configured(raw) { delete(s.feeds, raw) } }
	for raw := range s.status { if !s.configured(raw) { delete(s.status, raw) } }
	for raw := range s.inflight { if !s.configured(raw) { delete(s.inflight, raw) } }
}

// configured must be called while holding mu.
func (s *State) configured(raw string) bool {
	for _, value := range s.config.Values { if raw == value { return true } }
	return false
}

func (s *State) Feed(raw string) (domain.Feed, bool) {
	s.mu.RLock(); defer s.mu.RUnlock()
	f, ok := s.feeds[raw]
	return cloneFeed(f), ok
}

func (s *State) SetFeed(raw string, feed domain.Feed) {
	s.mu.Lock(); defer s.mu.Unlock()
	s.feeds[raw] = cloneFeed(feed)
}

// BeginFetch serializes updates per source, including watcher/timer overlap.
func (s *State) BeginFetch(raw string, at time.Time) (uint64, bool) {
	s.mu.Lock(); defer s.mu.Unlock()
	if !s.configured(raw) || s.inflight[raw] != 0 { return 0, false }
	s.nextTicket++
	s.inflight[raw] = s.nextTicket
	st := s.status[raw]; st.attempt = at.UTC().Format(time.RFC3339Nano); s.status[raw] = st
	return s.nextTicket, true
}

// CompleteFetch rejects results belonging to removed sources or superseded requests.
// Error details are deliberately not stored in the public view model.
func (s *State) CompleteFetch(raw string, ticket uint64, feed domain.Feed, at time.Time, fetchErr error) bool {
	s.mu.Lock(); defer s.mu.Unlock()
	if ticket == 0 || s.inflight[raw] != ticket || !s.configured(raw) { return false }
	delete(s.inflight, raw)
	st := s.status[raw]
	st.failed = fetchErr != nil
	if fetchErr == nil {
		stamp := at.UTC().Format(time.RFC3339Nano)
		old, ok := s.feeds[raw]
		if !ok || old.Title != feed.Title || old.Link != feed.Link || !reflect.DeepEqual(old.Items, feed.Items) { st.changed = stamp }
		st.success = stamp
		s.feeds[raw] = cloneFeed(feed)
	}
	s.status[raw] = st
	return true
}

// Feeds preserves the legacy endpoint contract (cached sources only).
func (s *State) Feeds() []domain.Feed {
	s.mu.RLock(); defer s.mu.RUnlock()
	feeds := make([]domain.Feed, 0, len(s.config.Values))
	for _, raw := range s.config.Values { if f, ok := s.feeds[raw]; ok { feeds = append(feeds, cloneFeed(f)) } }
	return feeds
}

func (s *State) Snapshot() domain.Snapshot {
	s.mu.RLock(); defer s.mu.RUnlock()
	out := domain.Snapshot{Version: 1, Title: s.config.WebTitle, Description: s.config.WebDes, AutoUpdatePush: s.config.AutoUpdatePush, ListHeight: s.config.ListHeight, Sources: make([]domain.Source, 0, len(s.config.Values))}
	seen := make(map[string]bool)
	for _, raw := range s.config.Values {
		if seen[raw] { continue }; seen[raw] = true
		f, cached := s.feeds[raw]
		st := s.status[raw]
		view := domain.Source{ID: stableID(raw), Title: f.Title, Link: homeLink(f.Link), Status: "loading", LastAttemptAt: st.attempt, LastSuccessAt: st.success, ContentChangedAt: st.changed, Items: make([]domain.Item, 0, len(f.Items))}
		if view.Title == "" {
			parsed, err := url.Parse(raw)
			if err == nil { view.Title = parsed.Hostname() }
			if view.Title == "" { view.Title = "订阅源" }
		}
		if cached { view.Status = "ready"; if len(f.Items) == 0 { view.Status = "empty" } }
		if st.failed {
			view.Status = "error"; view.Message = "获取失败，将在下一轮自动重试"
			if cached { view.Status = "stale"; view.Message = "同步失败，正在显示上次成功的内容" }
		}
		itemIDs := make(map[string]bool)
		for _, item := range f.Items {
			identity := item.GUID
			if identity == "" { identity = item.Link }
			if identity == "" { identity = item.Title }
			item.ID = stableID(view.ID + "\x00" + identity)
			if itemIDs[item.ID] { continue }; itemIDs[item.ID] = true
			item.Link = safeItemLink(item.Link)
			item.GUID = ""
			// Titles are rendered as text; upstream HTML is never sent to the reader.
			item.Description = ""
			view.Items = append(view.Items, item)
		}
		out.Sources = append(out.Sources, view)
	}
	data, _ := json.Marshal(out)
	out.Revision = stableID(string(data))
	return out
}

func stableID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

func homeLink(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") { return "" }
	return parsed.Scheme + "://" + parsed.Host
}

func safeItemLink(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") { return "" }
	parsed.User = nil
	q := parsed.Query()
	for key := range q {
		switch strings.ToLower(key) { case "auth", "token", "access_token", "password", "api_key", "apikey", "signature", "sign": q.Del(key) }
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func cloneFeed(feed domain.Feed) domain.Feed {
	cloned := feed
	cloned.Items = append([]domain.Item(nil), feed.Items...)
	if feed.Custom != nil { cloned.Custom = make(map[string]string, len(feed.Custom)); for k, v := range feed.Custom { cloned.Custom[k] = v } }
	return cloned
}

func cloneConfig(conf config.Config) config.Config {
	cloned := conf
	cloned.Values = append([]string(nil), conf.Values...)
	cloned.Keywords = append([]string(nil), conf.Keywords...)
	return cloned
}
