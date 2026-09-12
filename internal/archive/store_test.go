package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreLoadsAndMarksLinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archives.txt")
	if err := os.WriteFile(path, []byte("https://example.com/old\n"), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !store.Contains("https://example.com/old") {
		t.Fatal("existing archive link was not loaded")
	}

	isNew, err := store.MarkIfNew("https://example.com/new")
	if err != nil {
		t.Fatalf("MarkIfNew() error = %v", err)
	}
	if !isNew {
		t.Fatal("first MarkIfNew() = false, want true")
	}

	isNew, err = store.MarkIfNew("https://example.com/new")
	if err != nil {
		t.Fatalf("second MarkIfNew() error = %v", err)
	}
	if isNew {
		t.Fatal("second MarkIfNew() = true, want false")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if count := strings.Count(string(content), "https://example.com/new"); count != 1 {
		t.Fatalf("new link written %d times, want 1", count)
	}
}

func TestStoreReloadSwitchesArchive(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(first, []byte("https://example.com/first\n"), 0o600); err != nil {
		t.Fatalf("write first archive: %v", err)
	}
	if err := os.WriteFile(second, []byte("https://example.com/second\n"), 0o600); err != nil {
		t.Fatalf("write second archive: %v", err)
	}

	store, err := Open(first)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Reload(second); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if store.Contains("https://example.com/first") {
		t.Fatal("Reload() retained an entry from the old archive")
	}
	if !store.Contains("https://example.com/second") {
		t.Fatal("Reload() did not load the new archive")
	}
}
