package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mmcdole/gofeed"

	"rss-reader/models"
)

func TestNotifyRoutesToConfiguredProviders(t *testing.T) {
	tests := []struct {
		name         string
		route        string
		configure    func(baseURL string) models.Notify
		expectedPath string
	}{
		{
			name:  "feishu",
			route: FeiShuRoute,
			configure: func(baseURL string) models.Notify {
				return models.Notify{FeiShu: models.FeiShu{API: baseURL + "/feishu"}}
			},
			expectedPath: "/feishu",
		},
		{
			name:  "telegram",
			route: TelegramRoute,
			configure: func(baseURL string) models.Notify {
				return models.Notify{Telegram: models.Telegram{
					API:    baseURL + "/bot${token}/sendMessage",
					ChatId: "chat-id",
					Token:  "secret",
				}}
			},
			expectedPath: "/botsecret/sendMessage",
		},
		{
			name:  "dingtalk",
			route: DingtalkRoute,
			configure: func(baseURL string) models.Notify {
				return models.Notify{Dingtalk: models.Dingtalk{Webhook: baseURL + "/dingtalk"}}
			},
			expectedPath: "/dingtalk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodPost {
					t.Errorf("request method = %s, want POST", r.Method)
				}
				if r.URL.Path != tt.expectedPath {
					t.Errorf("request path = %q, want %q", r.URL.Path, tt.expectedPath)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()

			notifyConfig := tt.configure(server.URL)
			Notify(notifyConfig, Message{
				Routes:  []string{tt.route},
				Content: "test notification",
				FeedItem: gofeed.Item{
					Title: "Example",
					Link:  "https://example.com/item",
				},
			})

			if requests != 1 {
				t.Fatalf("provider received %d requests, want 1", requests)
			}
		})
	}
}
