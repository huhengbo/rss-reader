package models

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseConfFile(t *testing.T) {
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

		conf, err := ParseConfFile(path)
		if err != nil {
			t.Fatalf("ParseConfFile() error = %v", err)
		}
		if conf.Port != 8080 || conf.ReFresh != 5 || conf.WebTitle != "RSS Reader" {
			t.Fatalf("ParseConfFile() returned unexpected config: %#v", conf)
		}
		if !reflect.DeepEqual(conf.Values, []string{"https://example.com/feed.xml"}) {
			t.Fatalf("values = %#v", conf.Values)
		}
	})

	t.Run("returns error for invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"port":`), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		if _, err := ParseConfFile(path); err == nil {
			t.Fatal("ParseConfFile() error = nil, want non-nil")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.json")
		if _, err := ParseConfFile(path); err == nil {
			t.Fatal("ParseConfFile() error = nil, want non-nil")
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
