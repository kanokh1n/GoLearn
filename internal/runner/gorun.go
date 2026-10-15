// Package runner executes user solutions and checks them.
//
// Code runs directly on this machine without a sandbox. GoLearn is meant
// for personal use on localhost only.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GoTimeout limits how long `go run` / `go test` may take.
var GoTimeout = 20 * time.Second

// GoResult is the outcome of running Go code.
type GoResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
	TimedOut bool   `json:"timedOut"`
}

// RunGo compiles and runs code as package main and returns its combined output.
func RunGo(ctx context.Context, code string) (GoResult, error) {
	dir, err := goWorkspace(code, "")
	if err != nil {
		return GoResult{}, err
	}
	defer os.RemoveAll(dir)
	return goCmd(ctx, dir, "run", ".")
}

// TestGo runs the lesson tests against the user's code.
func TestGo(ctx context.Context, code, tests string) (GoResult, error) {
	dir, err := goWorkspace(code, tests)
	if err != nil {
		return GoResult{}, err
	}
	defer os.RemoveAll(dir)
	return goCmd(ctx, dir, "test", "-count=1", ".")
}

func goWorkspace(code, tests string) (string, error) {
	dir, err := os.MkdirTemp("", "golearn-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	files := map[string]string{
		"go.mod":  "module sandbox\n\ngo 1.22\n",
		"main.go": code,
	}
	if tests != "" {
		files["main_test.go"] = tests
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("write %s: %w", name, err)
		}
	}
	return dir, nil
}

func goCmd(ctx context.Context, dir string, args ...string) (GoResult, error) {
	ctx, cancel := context.WithTimeout(ctx, GoTimeout)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	res := GoResult{Output: cleanPaths(out.String(), dir)}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		res.TimedOut = true
		res.ExitCode = -1
		res.Output += fmt.Sprintf("\n[превышено время выполнения: %s]", GoTimeout)
		return res, nil
	}
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	case err != nil:
		return res, fmt.Errorf("run go %s: %w", strings.Join(args, " "), err)
	}
	return res, nil
}

// cleanPaths hides the temp dir from compiler errors: "/tmp/golearn-123/main.go:5" -> "main.go:5".
func cleanPaths(s, dir string) string {
	return strings.ReplaceAll(s, dir+string(filepath.Separator), "")
}
