package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"caipiao/backend/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL 或 DB_HOST 未配置")
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "用法: go run ./cmd/migrate [up|down|status|version]\n")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	if err := goose.RunContext(context.Background(), cmd, db, "migrations", args...); err != nil {
		log.Fatal(err)
	}
}
