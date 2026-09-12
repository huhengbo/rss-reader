package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"rss-reader/internal/config"
)

const (
	FeiShuRoute   = "feishu"
	DingtalkRoute = "dingding"
	TelegramRoute = "telegram"
	contentType   = "application/json"
	tokenReplace  = "${token}"
)

type Message struct {
	Routes   []string    `json:"routes"`
	Content  string      `json:"content"`
	FeedItem gofeed.Item `json:"feedItem"`
}

type feiShuMessage struct {
	MsgType string            `json:"msg_type"`
	Content feiShuMessageText `json:"content"`
}

type feiShuMessageText struct {
	Text string `json:"text"`
}

type telegramMessage struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

type dingtalkMessage struct {
	Msgtype string              `json:"msgtype"`
	Link    dingtalkMessageLink `json:"link"`
}

type dingtalkMessageLink struct {
	MessageUrl string `json:"messageUrl"`
	PicUrl     string `json:"picUrl"`
	Text       string `json:"text"`
	Title      string `json:"title"`
}

func Send(settings config.Notify, msg Message) {
	if len(msg.Routes) == 0 {
		return
	}
	for _, route := range msg.Routes {
		switch route {
		case FeiShuRoute:
			if settings.FeiShu.API != "" {
				sendToFeiShu(settings.FeiShu, msg)
			}
		case TelegramRoute:
			if settings.Telegram.Token != "" && settings.Telegram.ChatId != "" {
				time.Sleep(1500)
				sendToTelegram(settings.Telegram, msg)
			}
		case DingtalkRoute:
			if settings.Dingtalk.Webhook != "" {
				time.Sleep(1500)
				sendToDingtalk(settings.Dingtalk, msg)
			}
		default:
			log.Println("without route")
		}
	}
}

func sendToTelegram(settings config.Telegram, msg Message) {
	finalMsg, err := json.Marshal(telegramMessage{ChatId: settings.ChatId, Text: msg.Content})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	api := strings.ReplaceAll(settings.API, tokenReplace, settings.Token)
	requestPost(api, finalMsg)
}

func sendToDingtalk(settings config.Dingtalk, msg Message) {
	encodedSign := ""
	var timestamp int64
	if settings.Sign != "" {
		timestamp = time.Now().UnixNano() / int64(time.Millisecond)
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, settings.Sign)
		mac := hmac.New(sha256.New, []byte(settings.Sign))
		mac.Write([]byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		encodedSign = url.QueryEscape(sign)
	}

	finalMsg, err := json.Marshal(dingtalkMessage{
		Msgtype: "link",
		Link: dingtalkMessageLink{
			MessageUrl: msg.FeedItem.Link,
			Title:      msg.FeedItem.Title,
			Text:       msg.Content,
		},
	})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	api := settings.Webhook
	if encodedSign != "" {
		api = fmt.Sprintf("%s&timestamp=%d&sign=%s", api, timestamp, encodedSign)
	}
	requestPost(api, finalMsg)
}

func sendToFeiShu(settings config.FeiShu, msg Message) {
	finalMsg, err := json.Marshal(feiShuMessage{
		MsgType: "text",
		Content: feiShuMessageText{Text: msg.Content},
	})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	requestPost(settings.API, finalMsg)
}

func requestPost(url string, param []byte) {
	resp, err := http.Post(url, contentType, bytes.NewBuffer(param))
	if err != nil {
		log.Printf("http post err: %+v\n", err)
		return
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			log.Printf("http body close err: %+v\n", err)
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("http post read body err: %+v\n", err)
		return
	}
	log.Printf("response status: %s,response body:%s", string(body), resp.Status)
}
