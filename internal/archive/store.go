package archive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Store owns the persistent set of links that have already triggered notifications.
type Store struct {
	mu   sync.Mutex
	path string
	seen map[string]struct{}
}

func Open(path string) (*Store, error) {
	store := &Store{}
	if err := store.Reload(path); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Reload(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("archive path is empty")
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open archive %q: %w", path, err)
	}
	defer file.Close()

	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		link := strings.TrimSpace(scanner.Text())
		if link != "" {
			seen[link] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read archive %q: %w", path, err)
	}

	s.mu.Lock()
	s.path = path
	s.seen = seen
	s.mu.Unlock()
	return nil
}

func (s *Store) Contains(link string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[link]
	return ok
}

// MarkIfNew records link and returns true only for the first caller that sees it.
func (s *Store) MarkIfNew(link string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.seen[link]; ok {
		return false, nil
	}

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("open archive %q for append: %w", s.path, err)
	}
	if _, err := fmt.Fprintln(file, link); err != nil {
		_ = file.Close()
		return false, fmt.Errorf("append archive %q: %w", s.path, err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("close archive %q: %w", s.path, err)
	}

	s.seen[link] = struct{}{}
	return true, nil
}
