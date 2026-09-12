package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"RSS_READER_CONFIG",
		"RSS_READER_PORT",
		"RSS_READER_ARCHIVES",
		"RSS_READER_FEISHU_API",
		"RSS_READER_DINGTALK_WEBHOOK",
		"RSS_READER_DINGTALK_SIGN",
		"RSS_READER_TELEGRAM_API",
		"RSS_READER_TELEGRAM_CHAT_ID",
		"RSS_READER_TELEGRAM_TOKEN",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadFile(t *testing.T) {
	clearConfigEnv(t)

	t.Run("parses valid config", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		content := `{
  "port": 8080,
  "values": ["https://example.com/feed.xml"],
  "refresh": 5,
  "autoUpdatePush": 7,
  "listHeight": 600,
  "webTitle": "RSS Reader",
  "webDes": "Example",
  "keywords": ["cloudcone"],
  "archives": "archives.txt",
  "notify": {
    "feishu": {"api": ""},
    "dingtalk": {"webhook": "", "sign": ""},
    "telegram": {"api": "https://api.telegram.org/bot${token}/sendMessage", "chat_id": "", "token": ""}
  }
}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		conf, err := LoadFile(path)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		if conf.Port != 8080 || conf.ReFresh != 5 || conf.WebTitle != "RSS Reader" {
			t.Fatalf("LoadFile() returned unexpected config: %#v", conf)
		}
		if !reflect.DeepEqual(conf.Values, []string{"https://example.com/feed.xml"}) {
			t.Fatalf("values = %#v", conf.Values)
		}
	})

	t.Run("applies safe defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"values":[]}`), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		conf, err := LoadFile(path)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		if conf.Port != 8080 || conf.ReFresh != 5 || conf.ListHeight != 600 || conf.Archives != "archives.txt" {
			t.Fatalf("defaults not applied: %#v", conf)
		}
		if conf.Notify.Telegram.API != defaultTelegramAPI {
			t.Fatalf("telegram api = %q", conf.Notify.Telegram.API)
		}
	})

	t.Run("environment overrides sensitive settings", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"values":[]}`), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		t.Setenv("RSS_READER_PORT", "9090")
		t.Setenv("RSS_READER_ARCHIVES", "runtime-archives.txt")
		t.Setenv("RSS_READER_FEISHU_API", "https://example.test/feishu")
		t.Setenv("RSS_READER_DINGTALK_WEBHOOK", "https://example.test/dingtalk")
		t.Setenv("RSS_READER_DINGTALK_SIGN", "ding-secret")
		t.Setenv("RSS_READER_TELEGRAM_CHAT_ID", "chat")
		t.Setenv("RSS_READER_TELEGRAM_TOKEN", "token")

		conf, err := LoadFile(path)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		if conf.Port != 9090 || conf.Archives != "runtime-archives.txt" {
			t.Fatalf("runtime overrides not applied: %#v", conf)
		}
		if conf.Notify.FeiShu.API == "" || conf.Notify.Dingtalk.Sign != "ding-secret" || conf.Notify.Telegram.Token != "token" {
			t.Fatalf("secret overrides not applied: %#v", conf.Notify)
		}
	})

	t.Run("rejects partial telegram credentials", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"values":[]}`), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		t.Setenv("RSS_READER_TELEGRAM_TOKEN", "token-only")
		t.Setenv("RSS_READER_TELEGRAM_CHAT_ID", "")
		if _, err := LoadFile(path); err == nil {
			t.Fatal("LoadFile() error = nil, want non-nil")
		}
	})

	t.Run("returns error for invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"port":`), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		if _, err := LoadFile(path); err == nil {
			t.Fatal("LoadFile() error = nil, want non-nil")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.json")
		if _, err := LoadFile(path); err == nil {
			t.Fatal("LoadFile() error = nil, want non-nil")
		}
	})
}

func TestConfigGetIncrement(t *testing.T) {
	tests := []struct {
		name   string
		older  Config
		newer  Config
		wanted []string
	}{
		{
			name:   "returns newly added feeds",
			older:  Config{Values: []string{"https://example.com/a.xml", "https://example.com/b.xml"}},
			newer:  Config{Values: []string{"https://example.com/b.xml", "https://example.com/c.xml"}},
			wanted: []string{"https://example.com/c.xml"},
		},
		{
			name:   "returns empty when no feeds were added",
			older:  Config{Values: []string{"https://example.com/a.xml"}},
			newer:  Config{Values: []string{"https://example.com/a.xml"}},
			wanted: []string{},
		},
		{
			name:   "returns all feeds for an empty previous config",
			older:  Config{},
			newer:  Config{Values: []string{"https://example.com/a.xml", "https://example.com/b.xml"}},
			wanted: []string{"https://example.com/a.xml", "https://example.com/b.xml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.older.GetIncrement(tt.newer)
			if !reflect.DeepEqual(got, tt.wanted) {
				t.Fatalf("GetIncrement() = %#v, want %#v", got, tt.wanted)
			}
		})
	}
}
