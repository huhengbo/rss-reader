package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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

	maxAttempts         = 3
	notificationTimeout = 10 * time.Second
)

var defaultHTTPClient = newHTTPClient()

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

func Send(ctx context.Context, settings config.Notify, msg Message) {
	if len(msg.Routes) == 0 {
		return
	}

	for _, route := range msg.Routes {
		if err := sendRoute(ctx, settings, route, msg); err != nil {
			log.Printf("notify %s: %v", route, err)
		}
	}
}

func sendRoute(ctx context.Context, settings config.Notify, route string, msg Message) error {
	switch route {
	case FeiShuRoute:
		if settings.FeiShu.API == "" {
			return nil
		}
		return sendToFeiShu(ctx, settings.FeiShu, msg)
	case TelegramRoute:
		if settings.Telegram.Token == "" || settings.Telegram.ChatId == "" {
			return nil
		}
		return sendToTelegram(ctx, settings.Telegram, msg)
	case DingtalkRoute:
		if settings.Dingtalk.Webhook == "" {
			return nil
		}
		return sendToDingtalk(ctx, settings.Dingtalk, msg)
	default:
		return fmt.Errorf("unknown notification route %q", route)
	}
}

func sendToTelegram(ctx context.Context, settings config.Telegram, msg Message) error {
	finalMsg, err := json.Marshal(telegramMessage{ChatId: settings.ChatId, Text: msg.Content})
	if err != nil {
		return fmt.Errorf("marshal telegram message: %w", err)
	}
	api := strings.ReplaceAll(settings.API, tokenReplace, settings.Token)
	return requestPost(ctx, defaultHTTPClient, api, finalMsg)
}

func sendToDingtalk(ctx context.Context, settings config.Dingtalk, msg Message) error {
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
		return fmt.Errorf("marshal dingtalk message: %w", err)
	}

	api := settings.Webhook
	if encodedSign != "" {
		api = fmt.Sprintf("%s&timestamp=%d&sign=%s", api, timestamp, encodedSign)
	}
	return requestPost(ctx, defaultHTTPClient, api, finalMsg)
}

func sendToFeiShu(ctx context.Context, settings config.FeiShu, msg Message) error {
	finalMsg, err := json.Marshal(feiShuMessage{
		MsgType: "text",
		Content: feiShuMessageText{Text: msg.Content},
	})
	if err != nil {
		return fmt.Errorf("marshal feishu message: %w", err)
	}
	return requestPost(ctx, defaultHTTPClient, settings.API, finalMsg)
}

func requestPost(ctx context.Context, client *http.Client, endpoint string, param []byte) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(param))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			if attempt < maxAttempts {
				if err := waitRetry(ctx, attempt); err != nil {
					return err
				}
				continue
			}
			break
		}

		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32<<10))
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected response status %s", resp.Status)
		if !retryableStatus(resp.StatusCode) || attempt == maxAttempts {
			break
		}
		if err := waitRetry(ctx, attempt); err != nil {
			return err
		}
	}
	return lastErr
}

func waitRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt) * 250 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 5 * time.Second
	transport.IdleConnTimeout = 90 * time.Second
	return &http.Client{Transport: transport, Timeout: notificationTimeout}
}
