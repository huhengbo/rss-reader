package utils

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mmcdole/gofeed"

	"rss-reader/globals"
	"rss-reader/models"
)

func UpdateFeeds() {
	var (
		tick = time.Tick(time.Duration(globals.RssUrls.ReFresh) * time.Minute)
	)
	for {
		formattedTime := time.Now().Format("2006-01-02 15:04:05")
		for _, url := range globals.RssUrls.Values {
			go UpdateFeed(url, formattedTime)
		}
		<-tick
	}
}

func UpdateFeed(url, formattedTime string) {
	log.Printf("timer exec get: %s\n", url)
	result, err := globals.Fp.ParseURL(url)
	if err != nil {
		log.Printf("Error fetching feed: %v | %v", url, err)
		return
	}

	globals.Lock.RLock()
	cache, ok := globals.DbMap[url]
	globals.Lock.RUnlock()

	// feed内容无更新时无需更新缓存
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
	for _, v := range result.Items {
		customFeed.Items = append(customFeed.Items, models.Item{
			Link:        v.Link,
			Title:       v.Title,
			Description: v.Description,
		})
		Check(url, result, v)
	}
	globals.Lock.Lock()
	defer globals.Lock.Unlock()
	globals.DbMap[url] = customFeed
}

// GetFeeds 获取feeds列表
func GetFeeds() []models.Feed {
	feeds := make([]models.Feed, 0, len(globals.RssUrls.Values))
	for _, url := range globals.RssUrls.Values {
		globals.Lock.RLock()
		cache, ok := globals.DbMap[url]
		globals.Lock.RUnlock()
		if !ok {
			log.Printf("Error getting feed from db is null %v", url)
			continue
		}

		feeds = append(feeds, cache)
	}
	return feeds
}

func WatchConfigFileChanges(filePath string) {
	// 创建一个新的监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// 添加要监控的文件
	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}

	// 启动一个 goroutine 来处理文件变化事件
	go func() {
		for {
			time.Sleep(7 * time.Second)
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("通道关闭1")
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("文件已修改")
					globals.Init()
					formattedTime := time.Now().Format("2006-01-02 15:04:05")
					for _, url := range globals.RssUrls.Values {
						go UpdateFeed(url, formattedTime)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("通道关闭2")
					return
				}
				log.Println("错误:", err)
				return
			}
		}
	}()

	select {}
}

func Check(url string, result *gofeed.Feed, v *gofeed.Item) {
	if result == nil || v == nil || len(result.Items) == 0 {
		return
	}

	globals.Lock.RLock()
	cache, cacheOK := globals.DbMap[url]
	globals.Lock.RUnlock()

	if cacheOK && len(cache.Items) > 0 && cache.Items[0].Link == result.Items[0].Link {
		return
	}

	link := normalizeLink(v.Link)

	globals.Lock.RLock()
	_, fileCacheOK := globals.Hash[link]
	globals.Lock.RUnlock()
	if fileCacheOK {
		return
	}

	// 匹配关键词
	MatchStr(v.Title, func(msg string) {
		globals.Lock.Lock()
		if _, exists := globals.Hash[link]; exists {
			globals.Lock.Unlock()
			return
		}
		globals.Hash[link] = 1
		globals.Lock.Unlock()

		// 发送通知
		go Notify(Message{
			Routes:   []string{FeiShuRoute, TelegramRoute, DingtalkRoute},
			Content:  fmt.Sprintf("%s\n%s", msg, v.Link),
			FeedItem: *v,
		})
		globals.WriteFile(globals.RssUrls.Archives, link)
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
