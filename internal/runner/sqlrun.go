package runner

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure Go SQLite driver, no cgo needed
)

// SQLTimeout limits how long a single query may take.
var SQLTimeout = 5 * time.Second

// MaxRows caps the number of rows returned to the UI.
const MaxRows = 500

// Table is a query result or a table snapshot.
type Table struct {
	Name    string     `json:"name,omitempty"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// openDB creates a fresh in-memory database and applies setup.
func openDB(ctx context.Context, setup string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	// Each connection to :memory: is a separate database, so keep exactly one.
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, setup); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка подготовки базы урока: %w", err)
	}
	return db, nil
}

// RunSQL runs query against a fresh database built from setup.
func RunSQL(ctx context.Context, setup, query string) (Table, error) {
	ctx, cancel := context.WithTimeout(ctx, SQLTimeout)
	defer cancel()
	db, err := openDB(ctx, setup)
	if err != nil {
		return Table{}, err
	}
	defer db.Close()
	return queryTable(ctx, db, query)
}

// Tables returns a snapshot of every table created by setup, so the lesson
// page can show the data the user is querying.
func Tables(ctx context.Context, setup string) ([]Table, error) {
	ctx, cancel := context.WithTimeout(ctx, SQLTimeout)
	defer cancel()
	db, err := openDB(ctx, setup)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, n)
	}
	rows.Close()

	var tables []Table
	for _, n := range names {
		t, err := queryTable(ctx, db, fmt.Sprintf("SELECT * FROM %q", n))
		if err != nil {
			return nil, err
		}
		t.Name = n
		tables = append(tables, t)
	}
	return tables, nil
}

// CheckSQL runs the user's query and the reference solution and compares results.
// It returns the user's result, whether it matches, and a human-readable reason when it doesn't.
func CheckSQL(ctx context.Context, setup, query, solution string, ordered bool) (Table, bool, string, error) {
	got, err := RunSQL(ctx, setup, query)
	if err != nil {
		return Table{}, false, "", err
	}
	want, err := RunSQL(ctx, setup, solution)
	if err != nil {
		return got, false, "", fmt.Errorf("эталонное решение не выполнилось: %w", err)
	}
	ok, reason := sameResult(got, want, ordered)
	return got, ok, reason, nil
}

func sameResult(got, want Table, ordered bool) (bool, string) {
	if len(got.Columns) != len(want.Columns) {
		return false, fmt.Sprintf("Ожидалось столбцов: %d, получено: %d.", len(want.Columns), len(got.Columns))
	}
	if len(got.Rows) != len(want.Rows) {
		return false, fmt.Sprintf("Ожидалось строк: %d, получено: %d.", len(want.Rows), len(got.Rows))
	}
	g, w := joinRows(got.Rows), joinRows(want.Rows)
	if !ordered {
		sort.Strings(g)
		sort.Strings(w)
	}
	for i := range g {
		if g[i] != w[i] {
			if ordered {
				return false, "Строки отличаются от ожидаемых или идут в неправильном порядке."
			}
			return false, "Строки отличаются от ожидаемого результата."
		}
	}
	return true, ""
}

func joinRows(rows [][]string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.Join(r, "\x1f")
	}
	return out
}

func queryTable(ctx context.Context, db *sql.DB, query string) (Table, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return Table{}, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return Table{}, err
	}
	t := Table{Columns: cols, Rows: [][]string{}}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return Table{}, err
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = formatValue(v)
		}
		if len(t.Rows) < MaxRows {
			t.Rows = append(t.Rows, row)
		}
	}
	return t, rows.Err()
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	case float64:
		// Round so 2.5000000001 and 2.5 compare equal.
		s := fmt.Sprintf("%.4f", x)
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		return s
	default:
		return fmt.Sprint(x)
	}
}
