package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func main() {
	_ = godotenv.Load()
	pool, err := db.Connect(context.Background(), config.Load().DatabaseURL, 5, 1)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	ctx := context.Background()
	q := sqlcdb.New(pool)

	id := "ANN_mrbpmwm3ihqp"
	rows, err := pool.Query(ctx, `SELECT id, status, pinned FROM cms_announcements WHERE id = $1 OR pinned = true`, id)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("rows:")
	for rows.Next() {
		var rid, status string
		var pinned bool
		_ = rows.Scan(&rid, &status, &pinned)
		fmt.Printf("  id=%s status=%s pinned=%v\n", rid, status, pinned)
	}

	if err := q.ClearAnnouncementPinsAdmin(ctx); err != nil {
		fmt.Printf("ClearAnnouncementPinsAdmin err: %v\n", err)
		return
	}
	row, err := q.SetAnnouncementPinnedAdmin(ctx, sqlcdb.SetAnnouncementPinnedAdminParams{ID: id, Pinned: true})
	if err != nil {
		fmt.Printf("SetAnnouncementPinnedAdmin err: %T %v\n", err, err)
		fmt.Printf("is ErrNoRows: %v\n", err == pgx.ErrNoRows)
		return
	}
	fmt.Printf("pinned ok: id=%s status=%s pinned=%v\n", row.ID, row.Status, row.Pinned)
}
