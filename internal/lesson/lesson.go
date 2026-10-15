// Package lesson loads tracks and lessons from YAML files on disk.
package lesson

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind is the type of exercise a lesson contains.
type Kind string

const (
	KindGo  Kind = "go"
	KindSQL Kind = "sql"
)

// Lesson is a single exercise.
type Lesson struct {
	ID      string `yaml:"-"` // file name without extension, e.g. "01-hello"
	Title   string `yaml:"title"`
	Kind    Kind   `yaml:"type"`
	XP      int    `yaml:"xp"`
	Theory  string `yaml:"theory"` // Markdown
	Task    string `yaml:"task"`   // Markdown
	Hint    string `yaml:"hint"`
	Starter string `yaml:"starter"`

	// Go lessons: either ExpectedOutput or Tests must be set.
	ExpectedOutput string `yaml:"expected_output"`
	Tests          string `yaml:"tests"`

	// Reference solution. SQL lessons compare against it; for Go lessons it
	// is used by the content tests to prove the lesson is solvable.
	Solution string `yaml:"solution"`

	// SQL lessons.
	Setup   string `yaml:"setup"`   // overrides the track setup when set
	Ordered bool   `yaml:"ordered"` // row order matters

	Track *Track `yaml:"-"`
}

// Key uniquely identifies a lesson across tracks: "track/lesson".
func (l *Lesson) Key() string { return l.Track.ID + "/" + l.ID }

// SQLSetup returns the schema and data the lesson's query runs against.
func (l *Lesson) SQLSetup() string {
	if strings.TrimSpace(l.Setup) != "" {
		return l.Setup
	}
	return l.Track.Setup
}

// Track is an ordered group of lessons.
type Track struct {
	ID          string `yaml:"-"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Order       int    `yaml:"order"`
	Setup       string `yaml:"setup"` // shared SQL setup for SQL tracks

	Lessons []*Lesson `yaml:"-"`
}

// Next returns the lesson after l in the same track, or nil.
func (t *Track) Next(l *Lesson) *Lesson {
	for i, x := range t.Lessons {
		if x == l && i+1 < len(t.Lessons) {
			return t.Lessons[i+1]
		}
	}
	return nil
}

// Catalog holds all loaded tracks.
type Catalog struct {
	Tracks []*Track
	byKey  map[string]*Lesson
}

// Lesson finds a lesson by track and lesson id.
func (c *Catalog) Lesson(trackID, lessonID string) (*Lesson, bool) {
	l, ok := c.byKey[trackID+"/"+lessonID]
	return l, ok
}

// Load reads every track directory under root. A track directory contains
// track.yaml and one YAML file per lesson; lessons are sorted by file name.
func Load(root string) (*Catalog, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read lessons dir: %w", err)
	}
	c := &Catalog{byKey: map[string]*Lesson{}}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := loadTrack(filepath.Join(root, e.Name()), e.Name())
		if err != nil {
			return nil, err
		}
		c.Tracks = append(c.Tracks, t)
		for _, l := range t.Lessons {
			c.byKey[l.Key()] = l
		}
	}
	sort.SliceStable(c.Tracks, func(i, j int) bool { return c.Tracks[i].Order < c.Tracks[j].Order })
	return c, nil
}

func loadTrack(dir, id string) (*Track, error) {
	t := &Track{ID: id}
	if err := readYAML(filepath.Join(dir, "track.yaml"), t); err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		if name == "track" {
			continue
		}
		l := &Lesson{ID: name, Track: t}
		if err := readYAML(f, l); err != nil {
			return nil, err
		}
		if err := validate(l); err != nil {
			return nil, fmt.Errorf("lesson %s: %w", l.Key(), err)
		}
		if l.XP == 0 {
			l.XP = 10
		}
		t.Lessons = append(t.Lessons, l)
	}
	return t, nil
}

func validate(l *Lesson) error {
	switch l.Kind {
	case KindGo:
		if l.ExpectedOutput == "" && l.Tests == "" {
			return fmt.Errorf("go lesson needs expected_output or tests")
		}
	case KindSQL:
		if l.Solution == "" {
			return fmt.Errorf("sql lesson needs solution")
		}
		if strings.TrimSpace(l.SQLSetup()) == "" {
			return fmt.Errorf("sql lesson needs setup (in lesson or track)")
		}
	default:
		return fmt.Errorf("unknown type %q", l.Kind)
	}
	return nil
}

func readYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
