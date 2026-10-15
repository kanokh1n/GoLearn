package lesson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoad(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"b/track.yaml":     "title: B\norder: 1\n",
		"b/02-second.yaml": "title: Second\ntype: go\nexpected_output: hi\n",
		"b/01-first.yaml":  "title: First\ntype: go\ntests: x\nxp: 5\n",
		"a/track.yaml":     "title: A\norder: 2\nsetup: CREATE TABLE t(x);\n",
		"a/01-q.yaml":      "title: Q\ntype: sql\nsolution: SELECT 1\n",
		"notes.txt":        "ignored",
	})

	c, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Tracks) != 2 || c.Tracks[0].ID != "b" || c.Tracks[1].ID != "a" {
		t.Fatalf("tracks not sorted by order: %+v", c.Tracks)
	}
	b := c.Tracks[0]
	if b.Lessons[0].ID != "01-first" || b.Lessons[1].ID != "02-second" {
		t.Errorf("lessons not sorted by file name")
	}
	if b.Lessons[0].XP != 5 || b.Lessons[1].XP != 10 {
		t.Errorf("xp = %d, %d; want 5 and default 10", b.Lessons[0].XP, b.Lessons[1].XP)
	}
	if b.Next(b.Lessons[0]) != b.Lessons[1] || b.Next(b.Lessons[1]) != nil {
		t.Errorf("Next is wrong")
	}
	q, ok := c.Lesson("a", "01-q")
	if !ok || q.SQLSetup() != "CREATE TABLE t(x);" {
		t.Errorf("sql lesson should inherit track setup, got %+v", q)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := []struct {
		name, file, wantErr string
	}{
		{"unknown type", "title: X\ntype: rust\n", "unknown type"},
		{"go without check", "title: X\ntype: go\n", "expected_output or tests"},
		{"sql without solution", "title: X\ntype: sql\nsetup: x\n", "needs solution"},
		{"broken yaml", "title: [", "parse"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{
				"t/track.yaml": "title: T\n",
				"t/01-x.yaml":  tc.file,
			})
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
