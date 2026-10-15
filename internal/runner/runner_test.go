package runner

import (
	"context"
	"strings"
	"testing"
)

const setup = `
CREATE TABLE t (id INTEGER, name TEXT, score REAL);
INSERT INTO t VALUES (1, 'a', 2.5), (2, 'b', NULL), (3, 'c', 1.0);
`

func TestCheckSQL(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		query    string
		solution string
		ordered  bool
		wantOK   bool
		wantMsg  string
	}{
		{"same query", "SELECT * FROM t", "SELECT * FROM t", false, true, ""},
		{"different order is fine when unordered", "SELECT id FROM t ORDER BY id DESC", "SELECT id FROM t", false, true, ""},
		{"different order fails when ordered", "SELECT id FROM t ORDER BY id DESC", "SELECT id FROM t ORDER BY id", true, false, "порядке"},
		{"column count", "SELECT id, name FROM t", "SELECT id FROM t", false, false, "столбцов"},
		{"row count", "SELECT id FROM t WHERE id > 1", "SELECT id FROM t", false, false, "строк"},
		{"aliases do not matter", "SELECT id AS x FROM t", "SELECT id FROM t", false, true, ""},
		{"float rounding", "SELECT AVG(score) FROM t", "SELECT 1.75", false, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, msg, err := CheckSQL(ctx, setup, tc.query, tc.solution, tc.ordered)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.wantOK || !strings.Contains(msg, tc.wantMsg) {
				t.Errorf("ok=%v msg=%q, want ok=%v msg containing %q", ok, msg, tc.wantOK, tc.wantMsg)
			}
		})
	}
}

func TestCheckSQLSyntaxError(t *testing.T) {
	_, _, _, err := CheckSQL(context.Background(), setup, "SELEC * FROM t", "SELECT 1", false)
	if err == nil {
		t.Fatal("want error for invalid SQL")
	}
}

func TestTables(t *testing.T) {
	tables, err := Tables(context.Background(), setup)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || tables[0].Name != "t" || len(tables[0].Rows) != 3 {
		t.Fatalf("got %+v", tables)
	}
	if tables[0].Rows[1][2] != "NULL" {
		t.Errorf("NULL should be shown as NULL, got %q", tables[0].Rows[1][2])
	}
}

func TestRunGo(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles Go code")
	}
	ctx := context.Background()
	res, err := RunGo(ctx, "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hi\") }\n")
	if err != nil || res.ExitCode != 0 || res.Output != "hi\n" {
		t.Errorf("RunGo = %+v, %v", res, err)
	}

	res, err = RunGo(ctx, "package main\nfunc main() { x := 1 }\n")
	if err != nil || res.ExitCode == 0 || !strings.Contains(res.Output, "main.go") || strings.Contains(res.Output, "golearn-") {
		t.Errorf("compile error should be reported with a clean path: %+v, %v", res, err)
	}
}
