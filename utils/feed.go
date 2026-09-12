package utils

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mmcdole/gofeed"

	"rss-reader/internal/archive"
	appstate "rss-reader/internal/state"
	"rss-reader/models"
)

func UpdateFeeds(state *appstate.State, archiveStore *archive.Store) {
	config := state.Config()
	ticker := time.NewTicker(time.Duration(config.ReFresh) * time.Minute)
	defer ticker.Stop()

	for {
		formattedTime := time.Now().Format("2006-01-02 15:04:05")
		config = state.Config()
		for _, url := range config.Values {
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
	if ok &&
		len(result.Items) > 0 &&
		len(cache.Items) > 0 &&
		result.Items[0].Link == cache.Items[0].Link {
		return
	}

	customFeed := models.Feed{
		Title:  result.Title,
		Link:   result.Link,
		Custom: map[string]string{"lastupdate": formattedTime},
		Items:  make([]models.Item, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		customFeed.Items = append(customFeed.Items, models.Item{
			Link:        item.Link,
			Title:       item.Title,
			Description: item.Description,
		})
		Check(state, archiveStore, url, result, item)
	}
	state.SetFeed(url, customFeed)
}

func GetFeeds(state *appstate.State) []models.Feed {
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

			config, err := models.ParseConfFile(filePath)
			if err != nil {
				log.Printf("reload config: %v", err)
				continue
			}

			current := state.Config()
			if config.Archives != current.Archives {
				if err := archiveStore.Reload(config.Archives); err != nil {
					log.Printf("reload archive store: %v", err)
					continue
				}
			}

			state.ReplaceConfig(config)
			log.Println("configuration reloaded")

			formattedTime := time.Now().Format("2006-01-02 15:04:05")
			for _, url := range config.Values {
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

	config := state.Config()
	MatchStr(item.Title, config.Keywords, func(msg string) {
		isNew, err := archiveStore.MarkIfNew(link)
		if err != nil {
			log.Printf("record archive link: %v", err)
			return
		}
		if !isNew {
			return
		}

		go Notify(config.Notify, Message{
			Routes:   []string{FeiShuRoute, TelegramRoute, DingtalkRoute},
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
