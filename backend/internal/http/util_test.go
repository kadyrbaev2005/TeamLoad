package http

import (
	"context"
	"io"
	"strings"
	"testing"
)

func assertCount(t *testing.T, table string, want int) {
	t.Helper()
	var n int
	if err := testStore.Pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("%s: %d rows, want %d", table, n, want)
	}
}

func stringsReader(s string) io.Reader { return strings.NewReader(s) }

func lower(s string) string { return strings.ToLower(s) }
