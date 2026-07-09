package config

import "testing"

func TestBuildDatabaseURLFromParts(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "192.168.100.239")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "caipiao")
	t.Setenv("DB_USER", "caipiaoapp")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_SSLMODE", "disable")

	got := buildDatabaseURL()
	want := "postgres://caipiaoapp:secret@192.168.100.239:5432/caipiao?sslmode=disable"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildDatabaseURLPrefersDATABASE_URL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@host:5432/db?sslmode=require")
	t.Setenv("DB_HOST", "ignored")

	got := buildDatabaseURL()
	if got != "postgres://u:p@host:5432/db?sslmode=require" {
		t.Fatalf("got %q", got)
	}
}
