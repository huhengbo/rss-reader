package feed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mmcdole/gofeed"

	"rss-reader/internal/archive"
	"rss-reader/internal/config"
	"rss-reader/internal/domain"
	"rss-reader/internal/notify"
	appstate "rss-reader/internal/state"
)

const feedRequestTimeout = 15 * time.Second
var feedHTTPClient = &http.Client{Timeout: feedRequestTimeout}

func UpdateFeeds(ctx context.Context, state *appstate.State, archiveStore *archive.Store) {
	for {
		if ctx.Err() != nil { return }
		conf := state.Config()
		for _, raw := range conf.Values { go UpdateFeed(ctx, state, archiveStore, raw, "") }
		interval := time.Duration(conf.ReFresh) * time.Minute
		if interval <= 0 { interval = 5 * time.Minute }
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done(): timer.Stop(); return
		case <-timer.C:
		}
	}
}

// The final string parameter is retained for compatibility with existing callers.
// Timestamps are recorded at the actual attempt/success, always in UTC.
func UpdateFeed(ctx context.Context, state *appstate.State, archiveStore *archive.Store, feedURL, _ string) {
	if ctx.Err() != nil { return }
	ticket, ok := state.BeginFetch(feedURL, time.Now())
	if !ok { return }
	parser := gofeed.NewParser()
	parser.Client = feedHTTPClient
	result, err := parser.ParseURLWithContext(feedURL, ctx)
	if err != nil || result == nil {
		if err == nil { err = fmt.Errorf("empty parser result") }
		state.CompleteFetch(feedURL, ticket, domain.Feed{}, time.Now(), err)
		if ctx.Err() == nil { log.Printf("feed fetch failed (%T); retrying next cycle", err) }
		return
	}
	customFeed := domain.Feed{Title: result.Title, Link: result.Link, Custom: map[string]string{"lastupdate": time.Now().UTC().Format(time.RFC3339)}, Items: make([]domain.Item, 0, len(result.Items))}
	for _, item := range result.Items {
		if item == nil { continue }
		published := ""
		if item.PublishedParsed != nil { published = item.PublishedParsed.UTC().Format(time.RFC3339) }
		customFeed.Items = append(customFeed.Items, domain.Item{GUID: item.GUID, Link: item.Link, Title: item.Title, Description: item.Description, PublishedAt: published})
	}
	// A pinned first item cannot stand in for the entire feed's content.
	if !state.CompleteFetch(feedURL, ticket, customFeed, time.Now(), nil) { return }
	for _, item := range result.Items { Check(ctx, state, archiveStore, feedURL, result, item) }
}

func GetFeeds(state *appstate.State) []domain.Feed { return state.Feeds() }

func WatchConfigFileChanges(ctx context.Context, filePath string, state *appstate.State, archiveStore *archive.Store) {
	absolute, err := filepath.Abs(filePath)
	if err != nil { log.Printf("resolve configuration path: %v", err); return }
	watcher, err := fsnotify.NewWatcher()
	if err != nil { log.Printf("create config watcher: %v", err); return }
	defer watcher.Close()
	// Watch the directory so editor rename/atomic-save does not detach the watch.
	if err := watcher.Add(filepath.Dir(absolute)); err != nil { log.Printf("watch config directory: %v", err); return }
	var timer *time.Timer
	var reload <-chan time.Time
	defer func() { if timer != nil { timer.Stop() } }()
	for {
		select {
		case <-ctx.Done(): return
		case event, ok := <-watcher.Events:
			if !ok { return }
			if filepath.Clean(event.Name) != absolute || event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 { continue }
			if timer != nil { timer.Stop() }
			timer = time.NewTimer(150 * time.Millisecond)
			reload = timer.C
		case <-reload:
			reload = nil
			conf, err := config.LoadFile(absolute)
			if err != nil { log.Printf("configuration reload rejected (%T); retaining previous configuration", err); continue }
			current := state.Config()
			if conf.Port != current.Port { log.Print("port change requires a restart; retaining current port"); conf.Port = current.Port }
			if conf.Archives != current.Archives {
				if err := archiveStore.Reload(conf.Archives); err != nil { log.Printf("reload archive store: %v", err); continue }
			}
			state.ReplaceConfig(conf)
			log.Print("configuration reloaded")
			for _, raw := range conf.Values { go UpdateFeed(ctx, state, archiveStore, raw, "") }
		case err, ok := <-watcher.Errors:
			if !ok { return }
			log.Printf("config watcher error: %v", err)
		}
	}
}

func Check(ctx context.Context, state *appstate.State, archiveStore *archive.Store, _ string, result *gofeed.Feed, item *gofeed.Item) {
	if ctx.Err() != nil || result == nil || item == nil || len(result.Items) == 0 { return }
	link := normalizeLink(item.Link)
	if link == "" || archiveStore.Contains(link) { return }
	conf := state.Config()
	MatchTitle(item.Title, conf.Keywords, func(msg string) {
		isNew, err := archiveStore.MarkIfNew(link)
		if err != nil { log.Printf("record archive link: %v", err); return }
		if !isNew { return }
		go notify.Send(ctx, conf.Notify, notify.Message{Routes: []string{notify.FeiShuRoute, notify.TelegramRoute, notify.DingtalkRoute}, Content: fmt.Sprintf("%s\n%s", msg, item.Link), FeedItem: *item})
	})
}

func normalizeLink(link string) string {
	link = strings.TrimSpace(link)
	if index := strings.IndexByte(link, '?'); index >= 0 { link = link[:index] }
	if index := strings.IndexByte(link, '#'); index >= 0 { link = link[:index] }
	return link
}

func safeURLForLog(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil { return "<invalid-feed-url>" }
	parsed.User = nil; parsed.RawQuery = ""; parsed.Fragment = ""
	return parsed.String()
}
