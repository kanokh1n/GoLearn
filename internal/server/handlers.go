package server

import (
	"fmt"
	"net/http"

	"github.com/kanokh1n/GoLearn/internal/lesson"
	"github.com/kanokh1n/GoLearn/internal/runner"
)

type trackView struct {
	*lesson.Track
	Done    int
	Total   int
	Percent int
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	var tracks []trackView
	for _, t := range s.catalog.Tracks {
		v := trackView{Track: t, Total: len(t.Lessons)}
		for _, l := range t.Lessons {
			if s.progress.Done(l.Key()) {
				v.Done++
			}
		}
		if v.Total > 0 {
			v.Percent = v.Done * 100 / v.Total
		}
		tracks = append(tracks, v)
	}
	s.render(w, "index.html", map[string]any{
		"Title":  "GoLearn",
		"Tracks": tracks,
		"XP":     s.progress.TotalXP(),
	})
}

func (s *Server) lessonPage(w http.ResponseWriter, r *http.Request) {
	l, ok := s.catalog.Lesson(r.PathValue("track"), r.PathValue("lesson"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	code := l.Starter
	if d, ok := s.progress.Draft(l.Key()); ok {
		code = d
	}
	var tables []runner.Table
	if l.Kind == lesson.KindSQL {
		var err error
		if tables, err = runner.Tables(r.Context(), l.SQLSetup()); err != nil {
			s.log.Error("load tables", "lesson", l.Key(), "err", err)
		}
	}
	s.render(w, "lesson.html", map[string]any{
		"Title":  l.Title,
		"Lesson": l,
		"Code":   code,
		"Tables": tables,
		"Next":   l.Track.Next(l),
		"XP":     s.progress.TotalXP(),
	})
}

func (s *Server) saveDraft(l *lesson.Lesson, code string) {
	if err := s.progress.SaveDraft(l.Key(), code); err != nil {
		s.log.Error("save draft", "lesson", l.Key(), "err", err)
	}
}

func (s *Server) run(w http.ResponseWriter, r *http.Request, l *lesson.Lesson, code string) {
	s.saveDraft(l, code)
	switch l.Kind {
	case lesson.KindGo:
		res, err := runner.RunGo(r.Context(), code)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"output": res.Output, "exitCode": res.ExitCode})
	case lesson.KindSQL:
		t, err := runner.RunSQL(r.Context(), l.SQLSetup(), code)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"table": t})
	}
}

type checkResult struct {
	OK        bool          `json:"ok"`
	Message   string        `json:"message"`
	Output    string        `json:"output,omitempty"`
	Table     *runner.Table `json:"table,omitempty"`
	Error     string        `json:"error,omitempty"`
	XP        int           `json:"xp"`
	FirstTime bool          `json:"firstTime"`
	TotalXP   int           `json:"totalXp"`
	NextURL   string        `json:"nextUrl,omitempty"`
}

func (s *Server) check(w http.ResponseWriter, r *http.Request, l *lesson.Lesson, code string) {
	s.saveDraft(l, code)
	var res checkResult
	var err error
	switch l.Kind {
	case lesson.KindGo:
		res, err = checkGo(r, l, code)
	case lesson.KindSQL:
		res, err = checkSQL(r, l, code)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if res.OK {
		first, err := s.progress.Complete(l.Key(), l.XP)
		if err != nil {
			s.log.Error("save progress", "lesson", l.Key(), "err", err)
		}
		res.FirstTime = first
		res.XP = l.XP
		if next := l.Track.Next(l); next != nil {
			res.NextURL = "/lesson/" + l.Track.ID + "/" + next.ID
		} else {
			res.NextURL = "/"
		}
	}
	res.TotalXP = s.progress.TotalXP()
	writeJSON(w, http.StatusOK, res)
}

func checkGo(r *http.Request, l *lesson.Lesson, code string) (checkResult, error) {
	if l.Tests != "" {
		out, err := runner.TestGo(r.Context(), code, l.Tests)
		if err != nil {
			return checkResult{}, err
		}
		if out.ExitCode == 0 {
			return checkResult{OK: true, Message: "All tests passed!", Output: out.Output}, nil
		}
		return checkResult{Message: "Some tests failed. Read the output below.", Output: out.Output}, nil
	}

	out, err := runner.RunGo(r.Context(), code)
	if err != nil {
		return checkResult{}, err
	}
	if out.ExitCode != 0 {
		return checkResult{Message: "The program did not run successfully.", Output: out.Output}, nil
	}
	if normalizeOutput(out.Output) != normalizeOutput(l.ExpectedOutput) {
		return checkResult{
			Message: fmt.Sprintf("Wrong output.\nExpected:\n%s", normalizeOutput(l.ExpectedOutput)),
			Output:  out.Output,
		}, nil
	}
	return checkResult{OK: true, Message: "Correct!", Output: out.Output}, nil
}

func checkSQL(r *http.Request, l *lesson.Lesson, code string) (checkResult, error) {
	got, ok, reason, err := runner.CheckSQL(r.Context(), l.SQLSetup(), code, l.Solution, l.Ordered)
	if err != nil {
		// A syntax error in the user's query is a normal answer, not a server failure.
		return checkResult{Message: "Your query has an error.", Error: err.Error()}, nil
	}
	if !ok {
		return checkResult{Message: reason, Table: &got}, nil
	}
	return checkResult{OK: true, Message: "Correct!", Table: &got}, nil
}

func (s *Server) reset(w http.ResponseWriter, r *http.Request, l *lesson.Lesson, _ string) {
	if err := s.progress.ClearDraft(l.Key()); err != nil {
		s.log.Error("clear draft", "lesson", l.Key(), "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": l.Starter})
}
