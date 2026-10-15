# GoLearn

An interactive trainer for backend skills, in the spirit of SQL Academy and Codédex.
Lesson texts and the interface are in Russian, so it can be used for mentoring Russian-speaking students.
Every lesson has a short theory section and a task. You write the solution in the
built-in editor, press **Check**, and get the result right away.

Tracks right now:

- **Go Basics (Основы Go)**: 10 lessons, from Hello World to errors. Code runs with `go run` / `go test`.
- **SQL Basics (Основы SQL)**: 12 lessons, from `SELECT` to `JOIN` and subqueries. Queries run in SQLite and are compared with a reference query.

Progress, XP and your code drafts are saved in `data/progress.json`.

## Run

You need [Go](https://go.dev/dl/) 1.26 or newer.

```sh
git clone https://github.com/kanokh1n/GoLearn.git
cd GoLearn
make run        # or: go run ./cmd/golearn
```

Open http://127.0.0.1:7070.

Shortcuts in the editor: `Ctrl+Enter` runs the code, `Ctrl+Shift+Enter` checks it.

> ⚠️ Go lessons run your code directly on your machine, without a sandbox.
> GoLearn is meant for personal use, so keep it on `127.0.0.1`.

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
Tables come from `setup` in `track.yaml` (or in the lesson itself).

`go test ./lessons` checks that every solution passes and every starter does not.

## Roadmap

- Terminal lessons (Git, Linux, Docker) with a terminal inside the page
- More tracks: concurrency, HTTP APIs, Docker, Kubernetes, AWS
- Quizzes for theory topics

## Third-party code

[CodeMirror 5](https://codemirror.net/5/) (MIT) is vendored in `web/static/vendor/codemirror`.

## Troubleshooting

**The page shows something else (for example "Welcome to nginx!")** — another program already uses the port.
Pick a free one: `go run ./cmd/golearn -addr 127.0.0.1:9090`, then open http://127.0.0.1:9090.
