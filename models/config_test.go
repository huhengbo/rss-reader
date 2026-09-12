package models

import (
	"reflect"
	"testing"
)

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
