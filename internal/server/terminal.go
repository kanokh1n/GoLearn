package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/coder/websocket"
	"github.com/creack/pty"

	"github.com/kanokh1n/GoLearn/internal/lesson"
	"github.com/kanokh1n/GoLearn/internal/runner"
)

// terminalMsg is what the browser sends: keystrokes or a new window size.
type terminalMsg struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// terminal connects a browser terminal (xterm.js) to a bash shell running
// in the lesson workspace, over a WebSocket.
func (s *Server) terminal(w http.ResponseWriter, r *http.Request) {
	l, ok := s.catalog.Lesson(r.PathValue("track"), r.PathValue("lesson"))
	if !ok || l.Kind != lesson.KindShell {
		http.NotFound(w, r)
		return
	}
	dir, err := s.workspace(r.Context(), l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Accept checks that Origin matches Host, so other websites can't open a shell.
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()

	cmd := exec.Command("bash", "--noprofile", "--norc", "-i")
	cmd.Dir = dir
	cmd.Env = append(runner.ShellEnv(dir),
		"TERM=xterm-256color",
		`PS1=\[\e[36m\]\W\[\e[0m\] $ `,
		"HISTFILE=/dev/null",
	)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		conn.Close(websocket.StatusInternalError, "start shell failed")
		return
	}
	defer func() {
		ptmx.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// shell -> browser
	go func() {
		defer cancel()
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				conn.Close(websocket.StatusNormalClosure, "shell exited")
				return
			}
		}
	}()

	// browser -> shell
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) && websocket.CloseStatus(err) == -1 {
				s.log.Debug("terminal read", "err", err)
			}
			return
		}
		var msg terminalMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "input":
			if _, err := ptmx.Write([]byte(msg.Data)); err != nil {
				return
			}
		case "resize":
			if msg.Cols > 0 && msg.Rows > 0 {
				pty.Setsize(ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
			}
		}
	}
}

// workspace returns the lesson's working directory, creating it on first use.
func (s *Server) workspace(ctx context.Context, l *lesson.Lesson) (string, error) {
	dir := s.workspaceDir(l)
	if err := runner.EnsureWorkspace(ctx, dir, l.Setup); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Server) workspaceDir(l *lesson.Lesson) string {
	return filepath.Join(s.workspaceRoot, l.Track.ID, l.ID)
}
