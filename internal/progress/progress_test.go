package progress

import (
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "progress.json")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("missing file should not be an error: %v", err)
	}
	first, err := s.Complete("go/01", 10)
	if err != nil || !first {
		t.Fatalf("Complete = %v, %v", first, err)
	}
	if first, _ := s.Complete("go/01", 10); first {
		t.Error("second Complete of the same lesson should not count")
	}
	s.Complete("go/02", 15)
	s.SaveDraft("go/03", "package main")

	// Reopen from disk.
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Done("go/01") || s.Done("go/03") {
		t.Error("Done is wrong after reload")
	}
	if s.TotalXP() != 25 {
		t.Errorf("TotalXP = %d, want 25", s.TotalXP())
	}
	if d, ok := s.Draft("go/03"); !ok || d != "package main" {
		t.Errorf("Draft = %q, %v", d, ok)
	}
	s.ClearDraft("go/03")
	if _, ok := s.Draft("go/03"); ok {
		t.Error("draft should be cleared")
	}
}
