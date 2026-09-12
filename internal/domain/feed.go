package domain

// Feed retains the legacy /feeds and /ws representation.
type Feed struct {
	Title  string            `json:"title,omitempty"`
	Link   string            `json:"link"`
	Custom map[string]string `json:"custom,omitempty"`
	Items  []Item            `json:"items,omitempty"`
}

type Item struct {
	ID          string `json:"id,omitempty"`
	GUID        string `json:"-"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt,omitempty"`
}

// Snapshot is an authoritative, ordered replacement, including empty/failed sources.
// It deliberately does not contain runtime configuration or subscription URLs.
type Snapshot struct {
	Version        int      `json:"version"`
	Revision       string   `json:"revision"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	AutoUpdatePush int      `json:"autoUpdatePush"`
	ListHeight     int      `json:"listHeight"`
	Sources        []Source `json:"sources"`
}

type Source struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Link             string `json:"link"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	LastAttemptAt    string `json:"lastAttemptAt"`
	LastSuccessAt    string `json:"lastSuccessAt"`
	ContentChangedAt string `json:"contentChangedAt"`
	Items            []Item `json:"items"`
}
