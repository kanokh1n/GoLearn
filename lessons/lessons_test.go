// Package lessons_test checks the lesson content itself: every lesson loads,
// its reference solution passes, and its starter code (or untouched workspace) does not.
package lessons_test

import (
	"context"
	"strings"
	"testing"

	"github.com/kanokh1n/GoLearn/internal/lesson"
	"github.com/kanokh1n/GoLearn/internal/runner"
)

func TestLessons(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles every Go lesson")
	}
	c, err := lesson.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tr := range c.Tracks {
		for _, l := range tr.Lessons {
			t.Run(l.Key(), func(t *testing.T) {
				t.Parallel()
				if strings.TrimSpace(l.Solution) == "" {
					t.Fatal("lesson has no solution")
				}
				switch l.Kind {
				case lesson.KindGo:
					if !goPasses(t, ctx, l, l.Solution) {
						t.Error("solution does not pass")
					}
					if goPasses(t, ctx, l, l.Starter) {
						t.Error("starter code already passes")
					}
				case lesson.KindShell:
					dir := t.TempDir()
					if err := runner.PrepareWorkspace(ctx, dir, l.Setup); err != nil {
						t.Fatal(err)
					}
					if _, ok, err := runner.CheckShell(ctx, dir, l.Check); err != nil || ok {
						t.Fatalf("check passes before solving (err=%v)", err)
					}
					if out, ok, err := runner.RunScript(ctx, dir, l.Solution); err != nil || !ok {
						t.Fatalf("solution script failed: %v\n%s", err, out)
					}
					if out, ok, err := runner.CheckShell(ctx, dir, l.Check); err != nil || !ok {
						t.Errorf("check fails after solution: %v\n%s", err, out)
					}
				case lesson.KindSQL:
					if _, ok, msg, err := runner.CheckSQL(ctx, l.SQLSetup(), l.Solution, l.Solution, l.Ordered); err != nil || !ok {
						t.Errorf("solution fails: %v %s", err, msg)
					}
				}
			})
		}
	}
}

func goPasses(t *testing.T, ctx context.Context, l *lesson.Lesson, code string) bool {
	t.Helper()
	if l.Tests != "" {
		res, err := runner.TestGo(ctx, code, l.Tests)
		if err != nil {
			t.Fatal(err)
		}
		return res.ExitCode == 0
	}
	res, err := runner.RunGo(ctx, code)
	if err != nil {
		t.Fatal(err)
	}
	return res.ExitCode == 0 && strings.TrimSpace(res.Output) == strings.TrimSpace(l.ExpectedOutput)
}
