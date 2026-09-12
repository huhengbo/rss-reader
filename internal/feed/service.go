package feed

import (
	"fmt"
	"log"
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

func UpdateFeeds(state *appstate.State, archiveStore *archive.Store) {
	conf := state.Config()
	ticker := time.NewTicker(time.Duration(conf.ReFresh) * time.Minute)
	defer ticker.Stop()

	for {
		formattedTime := time.Now().Format("2006-01-02 15:04:05")
		conf = state.Config()
		for _, url := range conf.Values {
			go UpdateFeed(state, archiveStore, url, formattedTime)
		}
		<-ticker.C
	}
}

func UpdateFeed(state *appstate.State, archiveStore *archive.Store, url, formattedTime string) {
	log.Printf("timer exec get: %s\n", url)
	result, err := gofeed.NewParser().ParseURL(url)
	if err != nil {
		log.Printf("Error fetching feed: %v | %v", url, err)
		return
	}

	cache, ok := state.Feed(url)
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
		Check(state, archiveStore, url, result, item)
	}
	state.SetFeed(url, customFeed)
}

func GetFeeds(state *appstate.State) []domain.Feed {
	return state.Feeds()
}

func WatchConfigFileChanges(filePath string, state *appstate.State, archiveStore *archive.Store) {
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
			for _, url := range conf.Values {
				go UpdateFeed(state, archiveStore, url, formattedTime)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		}
	}
}

func Check(state *appstate.State, archiveStore *archive.Store, url string, result *gofeed.Feed, item *gofeed.Item) {
	if result == nil || item == nil || len(result.Items) == 0 {
		return
	}

	cache, cacheOK := state.Feed(url)
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

		go notify.Send(conf.Notify, notify.Message{
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
