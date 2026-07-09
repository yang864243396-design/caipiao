package main

import (
	"context"
	"encoding/json"
	"fmt"

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
	row, err := sqlcdb.New(pool).GetMemberProfileByAccount(context.Background(), "vs8888")
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(map[string]any{
		"memberId": row.ID,
		"account":  row.Account,
	}, "", "  ")
	fmt.Println(string(b))
}
