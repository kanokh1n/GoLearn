// Package server is the HTTP layer: pages and the run/check JSON API.
package server

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/kanokh1n/GoLearn/internal/lesson"
	"github.com/kanokh1n/GoLearn/internal/progress"
	"github.com/kanokh1n/GoLearn/web"
)

// Server serves the GoLearn UI.
type Server struct {
	catalog  *lesson.Catalog
	progress *progress.Store
	log      *slog.Logger
	pages    map[string]*template.Template
	md       goldmark.Markdown

	workspaceRoot string // shell lessons get a folder here: <root>/<track>/<lesson>
}

// New parses templates and builds a Server.
func New(c *lesson.Catalog, p *progress.Store, workspaceRoot string, log *slog.Logger) (*Server, error) {
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}
	s := &Server{
		catalog:       c,
		progress:      p,
		log:           log,
		pages:         map[string]*template.Template{},
		md:            goldmark.New(goldmark.WithExtensions(extension.GFM)),
		workspaceRoot: root,
	}
	funcs := template.FuncMap{"markdown": s.markdown, "done": p.Done}
	for _, page := range []string{"index.html", "lesson.html"} {
		t, err := template.New("").Funcs(funcs).ParseFS(web.FS, "templates/layout.html", "templates/"+page)
		if err != nil {
			return nil, err
		}
		s.pages[page] = t
	}
	return s, nil
}

// Handler returns the router with logging middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(web.FS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /lesson/{track}/{lesson}", s.lessonPage)
	mux.HandleFunc("POST /api/lesson/{track}/{lesson}/run", s.withLesson(s.run))
	mux.HandleFunc("POST /api/lesson/{track}/{lesson}/check", s.withLesson(s.check))
	mux.HandleFunc("POST /api/lesson/{track}/{lesson}/reset", s.withLesson(s.reset))
	mux.HandleFunc("GET /api/lesson/{track}/{lesson}/terminal", s.terminal)
	return s.logRequests(localOnly(mux))
}

// localOnly blocks requests that come from other websites. GoLearn runs code
// and opens shells on this machine, so a page on the internet must not be able
// to call its API from the user's browser (CSRF) or reach it via DNS rebinding.
func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			http.Error(w, "GoLearn accepts only localhost requests", http.StatusForbidden)
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
			http.Error(w, "cross-site request blocked", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost {
			if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
				http.Error(w, "cross-origin request blocked", http.StatusForbidden)
				return
			}
			// Forms can't send application/json, so this also stops form-based CSRF.
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "took", time.Since(start).Round(time.Millisecond))
	})
}

func (s *Server) markdown(src string) template.HTML {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	// Lesson files are written by us, so their HTML is trusted.
	return template.HTML(buf.String())
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	var buf bytes.Buffer
	if err := s.pages[page].ExecuteTemplate(&buf, "layout", data); err != nil {
		s.log.Error("render", "page", page, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

type lessonHandler func(w http.ResponseWriter, r *http.Request, l *lesson.Lesson, code string)

// withLesson resolves the lesson from the path and decodes {"code": "..."} from the body.
func (s *Server) withLesson(h lessonHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l, ok := s.catalog.Lesson(r.PathValue("track"), r.PathValue("lesson"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "урок не найден"})
			return
		}
		var body struct {
			Code string `json:"code"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректное тело запроса"})
				return
			}
		}
		h(w, r, l, body.Code)
	}
}

// normalizeOutput ignores trailing spaces and Windows line endings.
func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
