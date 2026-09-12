package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mmcdole/gofeed"

	"rss-reader/internal/config"
)

func TestSendRoutesToConfiguredProviders(t *testing.T) {
	tests := []struct {
		name         string
		route        string
		configure    func(baseURL string) config.Notify
		expectedPath string
	}{
		{
			name:  "feishu",
			route: FeiShuRoute,
			configure: func(baseURL string) config.Notify {
				return config.Notify{FeiShu: config.FeiShu{API: baseURL + "/feishu"}}
			},
			expectedPath: "/feishu",
		},
		{
			name:  "telegram",
			route: TelegramRoute,
			configure: func(baseURL string) config.Notify {
				return config.Notify{Telegram: config.Telegram{
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
			configure: func(baseURL string) config.Notify {
				return config.Notify{Dingtalk: config.Dingtalk{Webhook: baseURL + "/dingtalk"}}
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
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			Send(context.Background(), tt.configure(server.URL), Message{
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

func TestRequestPostRetriesRetryableStatus(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := requestPost(context.Background(), server.Client(), server.URL, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("requestPost() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestRequestPostHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := requestPost(ctx, defaultHTTPClient, "https://example.invalid", []byte(`{}`))
	if err == nil {
		t.Fatal("requestPost() error = nil, want context cancellation")
	}
}
