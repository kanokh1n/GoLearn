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

// ShellTimeout limits setup and check scripts.
var ShellTimeout = 15 * time.Second

// ShellEnv is the environment for lesson scripts and the lesson terminal.
// Git identity is fixed so commits work even without a global git config.
func ShellEnv(workspace string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "LC_ALL=") { // LC_ALL would override LC_COLLATE below
			env = append(env, kv)
		}
	}
	return append(env,
		"GOLEARN_WORKSPACE="+workspace,
		// Workspaces live inside the GoLearn checkout; without this, git commands in a
		// lesson that has no repo yet would find and change GoLearn's own repository.
		"GIT_CEILING_DIRECTORIES="+filepath.Dir(workspace),
		"GIT_AUTHOR_NAME=GoLearn Student",
		"GIT_AUTHOR_EMAIL=student@golearn.local",
		"GIT_COMMITTER_NAME=GoLearn Student",
		"GIT_COMMITTER_EMAIL=student@golearn.local",
		"GIT_CONFIG_GLOBAL=/dev/null", // user's aliases and hooks must not change lesson results
		"GIT_EDITOR=true",             // `git merge` / `git revert` must not open an editor and hang
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=init.defaultBranch",
		"GIT_CONFIG_VALUE_0=main",
		"LC_COLLATE=C", // `sort` must order the same way in the terminal and in checks
	)
}

// PrepareWorkspace recreates dir from scratch and runs the lesson setup script in it.
func PrepareWorkspace(ctx context.Context, dir, setup string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clean workspace: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	if setup == "" {
		return nil
	}
	out, ok, err := runBash(ctx, dir, setup)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("lesson setup failed:\n%s", out)
	}
	return nil
}

// EnsureWorkspace prepares the workspace only if it does not exist yet,
// so reloading the page keeps the user's work.
func EnsureWorkspace(ctx context.Context, dir, setup string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	return PrepareWorkspace(ctx, dir, setup)
}

// CheckShell runs the lesson check script in the workspace.
func CheckShell(ctx context.Context, dir, check string) (output string, ok bool, err error) {
	return runBash(ctx, dir, check)
}

// RunScript runs any bash script in the workspace (used by content tests to apply solutions).
func RunScript(ctx context.Context, dir, script string) (string, bool, error) {
	return runBash(ctx, dir, script)
}

func runBash(ctx context.Context, dir, script string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, ShellTimeout)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = ShellEnv(dir)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out.String() + "\n[превышено время выполнения]", false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return out.String(), false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("run bash: %w", err)
	}
	return out.String(), true, nil
}
