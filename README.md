# GoLearn

An interactive trainer for backend skills, in the spirit of SQL Academy and Codédex.
Lesson texts and the interface are in Russian, so it can be used for mentoring Russian-speaking students.
Every lesson has a short theory section and a task. You write the solution in the
built-in editor, press **Check**, and get the result right away.

Tracks right now:

- **Go Basics (Основы Go)**: 10 lessons, from Hello World to errors. Code runs with `go run` / `go test`.
- **SQL Basics (Основы SQL)**: 12 lessons, from `SELECT` to `JOIN` and subqueries. Queries run in SQLite and are compared with a reference query.

- **Linux и терминал**: 8 lessons: files, `find`, `grep`, pipes, permissions, bash scripts.
- **Git**: 8 lessons, from `git init` to branches, merge conflicts, `.gitignore` and `revert`.

Terminal lessons open a real bash shell in the browser (xterm.js + WebSocket + PTY).
Each lesson gets its own folder in `data/workspaces/`, and a check script verifies the result.

Progress, XP and your code drafts are saved in `data/progress.json`.

## Run

You need [Go](https://go.dev/dl/) 1.21+ (it downloads the 1.26 toolchain itself), plus `bash` and `git` for terminal lessons.
Works on macOS and Linux (terminal lessons don't work on Windows; use WSL).

```sh
git clone https://github.com/kanokh1n/GoLearn.git
cd GoLearn
make run        # or: go run ./cmd/golearn
```

Open http://127.0.0.1:7070.

Shortcuts in the editor: `Ctrl+Enter` runs the code, `Ctrl+Shift+Enter` checks it.

> ⚠️ Lessons run your code and a shell directly on your machine, without a sandbox.
> GoLearn is meant for personal use, so keep it on `127.0.0.1`. The server rejects
> requests from other websites and non-localhost hosts.

## Project structure

```
cmd/golearn/        entry point: flags, HTTP server, graceful shutdown
internal/lesson/    loads tracks and lessons from YAML
internal/runner/    runs Go code and SQL queries, compares results
internal/progress/  completed lessons, XP and drafts in a JSON file
internal/server/    pages and the run/check JSON API
web/                HTML templates, CSS, JS (embedded into the binary)
lessons/            lesson content, one folder per track
```

## Adding a lesson

Create a YAML file in a track folder. Files are sorted by name, so use a number prefix.

Go lesson, checked by output:

```yaml
title: Hello, World!
type: go
xp: 10
theory: |
  Markdown text...
task: |
  Print `Hello, World!`
hint: |
  Use fmt.Println
starter: |
  package main
  ...
expected_output: |
  Hello, World!
solution: |
  package main
  ...
```

Go lesson checked by tests: replace `expected_output` with `tests:` (a `package main` test file).

SQL lesson: `type: sql`, a `solution` query, and optionally `ordered: true` when row order matters.

Terminal lesson: `type: shell`, a `setup` bash script that prepares the workspace, a `check`
bash script (exit code 0 means solved, its output is shown to the student) and a `solution` script.
Tables come from `setup` in `track.yaml` (or in the lesson itself).

`go test ./lessons` checks that every solution passes and every starter does not.

## Roadmap

- Docker lessons in the terminal
- More tracks: concurrency, HTTP APIs, Docker, Kubernetes, AWS
- Quizzes for theory topics

## Third-party code

[CodeMirror 5](https://codemirror.net/5/) and [xterm.js](https://xtermjs.org/) (both MIT) are vendored in `web/static/vendor/`.

## Troubleshooting

**The page shows something else (for example "Welcome to nginx!")** — another program already uses the port.
Pick a free one: `go run ./cmd/golearn -addr 127.0.0.1:9090`, then open http://127.0.0.1:9090.
