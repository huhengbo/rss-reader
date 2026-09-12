package utils

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

	"rss-reader/models"
)

const (
	FeiShuRoute   = "feishu"
	DingtalkRoute = "dingding"
	TelegramRoute = "telegram"
	ContentType   = "application/json"
	TokenReplace  = "${token}"
)

type Message struct {
	Routes   []string    `json:"routes"`
	Content  string      `json:"content"`
	FeedItem gofeed.Item `json:"feedItem"`
}

type FeiShuMessage struct {
	MsgType string            `json:"msg_type"`
	Content FeiShuMessageText `json:"content"`
}

type FeiShuMessageText struct {
	Text string `json:"text"`
}

type TelegramMessage struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

type DingtalkMessage struct {
	Msgtype string              `json:"msgtype"`
	Link    DingtalkMessageLink `json:"link"`
}

type DingtalkMessageLink struct {
	MessageUrl string `json:"messageUrl"`
	PicUrl     string `json:"picUrl"`
	Text       string `json:"text"`
	Title      string `json:"title"`
}

func Notify(config models.Notify, msg Message) {
	if len(msg.Routes) == 0 {
		return
	}
	for _, route := range msg.Routes {
		switch route {
		case FeiShuRoute:
			if config.FeiShu.API != "" {
				sendToFeiShu(config.FeiShu, msg)
			}
		case TelegramRoute:
			if config.Telegram.Token != "" && config.Telegram.ChatId != "" {
				time.Sleep(1500)
				sendToTelegram(config.Telegram, msg)
			}
		case DingtalkRoute:
			if config.Dingtalk.Webhook != "" {
				time.Sleep(1500)
				sendToDingtalk(config.Dingtalk, msg)
			}
		default:
			log.Println("without route")
		}
	}
}

func sendToTelegram(config models.Telegram, msg Message) {
	finalMsg, err := json.Marshal(
		TelegramMessage{
			ChatId: config.ChatId,
			Text:   msg.Content,
		})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	api := strings.ReplaceAll(config.API, TokenReplace, config.Token)
	requestPost(api, finalMsg)
}

func sendToDingtalk(config models.Dingtalk, msg Message) {
	encodedSign := ""
	var timestamp int64
	if config.Sign != "" {
		timestamp = time.Now().UnixNano() / int64(time.Millisecond)
		secret := config.Sign
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		signData := mac.Sum(nil)
		sign := base64.StdEncoding.EncodeToString(signData)
		encodedSign = url.QueryEscape(sign)
	}

	finalMsg, err := json.Marshal(
		DingtalkMessage{
			Msgtype: "link",
			Link: DingtalkMessageLink{
				MessageUrl: msg.FeedItem.Link,
				Title:      msg.FeedItem.Title,
				Text:       msg.Content,
			},
		})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	api := config.Webhook
	if encodedSign != "" {
		api = fmt.Sprintf("%s&timestamp=%d&sign=%s", api, timestamp, encodedSign)
	}

	requestPost(api, finalMsg)
}

func sendToFeiShu(config models.FeiShu, msg Message) {
	finalMsg, err := json.Marshal(
		FeiShuMessage{
			MsgType: "text",
			Content: FeiShuMessageText{
				Text: msg.Content,
			},
		})
	if err != nil {
		log.Printf("json marshal err: %+v\n", err)
		return
	}
	requestPost(config.API, finalMsg)
}

func requestPost(url string, param []byte) {
	requestBody := bytes.NewBuffer(param)
	resp, err := http.Post(url, ContentType, requestBody)

	if err != nil {
		log.Printf("http post err: %+v\n", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
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
