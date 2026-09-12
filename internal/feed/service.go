package feed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
		formattedTime := time.Now().Format("2006-01-02 15:04:05")
		conf := state.Config()
		for _, feedURL := range conf.Values {
			go UpdateFeed(ctx, state, archiveStore, feedURL, formattedTime)
		}

		timer := time.NewTimer(time.Duration(conf.ReFresh) * time.Minute)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func UpdateFeed(ctx context.Context, state *appstate.State, archiveStore *archive.Store, feedURL, formattedTime string) {
	log.Printf("timer exec get: %s\n", safeURLForLog(feedURL))

	parser := gofeed.NewParser()
	parser.Client = feedHTTPClient
	result, err := parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("Error fetching feed: %s | %v", safeURLForLog(feedURL), err)
		}
		return
	}

	cache, ok := state.Feed(feedURL)
	if ok && len(result.Items) > 0 && len(cache.Items) > 0 && result.Items[0].Link == cache.Items[0].Link {
		return
	}

	customFeed := domain.Feed{
		Title:  result.Title,
		Link:   result.Link,
		Custom: map[string]string{"lastupdate": formattedTime},
		Items:  make([]domain.Item, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		customFeed.Items = append(customFeed.Items, domain.Item{
			Link:        item.Link,
			Title:       item.Title,
			Description: item.Description,
		})
		Check(ctx, state, archiveStore, feedURL, result, item)
	}
	state.SetFeed(feedURL, customFeed)
}

func GetFeeds(state *appstate.State) []domain.Feed {
	return state.Feeds()
}

func WatchConfigFileChanges(ctx context.Context, filePath string, state *appstate.State, archiveStore *archive.Store) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("create config watcher: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(filePath); err != nil {
		log.Printf("watch config file %q: %v", filePath, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			conf, err := config.LoadFile(filePath)
			if err != nil {
				log.Printf("reload config: %v", err)
				continue
			}

			current := state.Config()
			if conf.Archives != current.Archives {
				if err := archiveStore.Reload(conf.Archives); err != nil {
					log.Printf("reload archive store: %v", err)
					continue
				}
			}

			state.ReplaceConfig(conf)
			log.Println("configuration reloaded")

			formattedTime := time.Now().Format("2006-01-02 15:04:05")
			for _, feedURL := range conf.Values {
				go UpdateFeed(ctx, state, archiveStore, feedURL, formattedTime)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		}
	}
}

func Check(ctx context.Context, state *appstate.State, archiveStore *archive.Store, feedURL string, result *gofeed.Feed, item *gofeed.Item) {
	if result == nil || item == nil || len(result.Items) == 0 {
		return
	}

	cache, cacheOK := state.Feed(feedURL)
	if cacheOK && len(cache.Items) > 0 && cache.Items[0].Link == result.Items[0].Link {
		return
	}

	link := normalizeLink(item.Link)
	if archiveStore.Contains(link) {
		return
	}

	conf := state.Config()
	MatchTitle(item.Title, conf.Keywords, func(msg string) {
		isNew, err := archiveStore.MarkIfNew(link)
		if err != nil {
			log.Printf("record archive link: %v", err)
			return
		}
		if !isNew {
			return
		}

		go notify.Send(ctx, conf.Notify, notify.Message{
			Routes:   []string{notify.FeiShuRoute, notify.TelegramRoute, notify.DingtalkRoute},
			Content:  fmt.Sprintf("%s\n%s", msg, item.Link),
			FeedItem: *item,
		})
	})
}

func normalizeLink(link string) string {
	link = strings.TrimSpace(link)
	if index := strings.IndexByte(link, '?'); index >= 0 {
		link = link[:index]
	}
	if index := strings.IndexByte(link, '#'); index >= 0 {
		link = link[:index]
	}
	return link
}

func safeURLForLog(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<invalid-feed-url>"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
