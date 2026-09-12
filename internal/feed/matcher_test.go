package feed

import "testing"

func TestMatchTitle(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		input    string
		wantHits int
	}{
		{
			name:     "matches keyword case insensitively",
			patterns: []string{"CloudCone"},
			input:    "cloudcone restock available",
			wantHits: 1,
		},
		{
			name:     "matches include keyword when exclusion is absent",
			patterns: []string{"抽奖 -测评"},
			input:    "新的抽奖活动",
			wantHits: 1,
		},
		{
			name:     "exclusion keyword blocks match",
			patterns: []string{"抽奖 -测评"},
			input:    "抽奖测评活动",
			wantHits: 0,
		},
		{
			name:     "unrelated title does not match",
			patterns: []string{"racknerd"},
			input:    "unrelated feed item",
			wantHits: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := 0
			MatchTitle(tt.input, tt.patterns, func(string) {
				hits++
			})
			if hits != tt.wantHits {
				t.Fatalf("MatchTitle() callback count = %d, want %d", hits, tt.wantHits)
			}
		})
	}
}
