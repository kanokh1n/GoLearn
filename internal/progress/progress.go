// Package progress stores completed lessons, XP and code drafts in a JSON file.
package progress

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Completion records when a lesson was solved and how much XP it gave.
type Completion struct {
	XP int       `json:"xp"`
	At time.Time `json:"at"`
}

type data struct {
	Completed map[string]Completion `json:"completed"`
	Drafts    map[string]string     `json:"drafts"`
}

// Store is safe for concurrent use.
type Store struct {
	mu   sync.RWMutex
	path string
	d    data
}

// Open loads the progress file. A missing file means no progress yet.
func Open(path string) (*Store, error) {
	s := &Store{path: path, d: data{Completed: map[string]Completion{}, Drafts: map[string]string{}}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read progress: %w", err)
	}
	if err := json.Unmarshal(b, &s.d); err != nil {
		return nil, fmt.Errorf("parse progress %s: %w", path, err)
	}
	if s.d.Completed == nil {
		s.d.Completed = map[string]Completion{}
	}
	if s.d.Drafts == nil {
		s.d.Drafts = map[string]string{}
	}
	return s, nil
}

// Done reports whether the lesson is completed.
func (s *Store) Done(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.d.Completed[key]
	return ok
}

// TotalXP sums XP over all completed lessons.
func (s *Store) TotalXP() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, c := range s.d.Completed {
		total += c.XP
	}
	return total
}

// Draft returns the last code the user wrote for a lesson.
func (s *Store) Draft(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.d.Drafts[key]
	return d, ok
}

// SaveDraft remembers the user's code for a lesson.
func (s *Store) SaveDraft(key, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.Drafts[key] = code
	return s.save()
}

// ClearDraft forgets the user's code, so the starter code shows again.
func (s *Store) ClearDraft(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.d.Drafts, key)
	return s.save()
}

// Complete marks a lesson as solved. It returns true if this is the first time.
func (s *Store) Complete(key string, xp int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.d.Completed[key]; ok {
		return false, nil
	}
	s.d.Completed[key] = Completion{XP: xp, At: time.Now()}
	return true, s.save()
}

// save writes atomically: temp file + rename, so a crash never leaves a half-written file.
func (s *Store) save() error {
	b, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write progress: %w", err)
	}
	return os.Rename(tmp, s.path)
}
