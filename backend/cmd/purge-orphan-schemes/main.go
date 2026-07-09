// 预览或删除全站孤儿方案定义（scheme_definitions 无对应 scheme_instances）。
// go run ./cmd/purge-orphan-schemes           # 仅统计
// go run ./cmd/purge-orphan-schemes -apply    # 执行删除
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

const orphanSQL = `
SELECT d.id, m.account, d.scheme_name, d.kind, d.lottery_code, d.created_at
FROM scheme_definitions d
JOIN members m ON m.id = d.member_id
WHERE NOT EXISTS (SELECT 1 FROM scheme_instances i WHERE i.definition_id = d.id)
ORDER BY d.created_at`

func main() {
	_ = godotenv.Load()
	apply := flag.Bool("apply", false, "执行删除（默认仅预览）")
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, orphanSQL)
	if err != nil {
		fmt.Println("query:", err)
		os.Exit(1)
	}
	defer rows.Close()

	type row struct {
		id, account, name, kind, lottery string
		createdAt                        interface{}
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.account, &r.name, &r.kind, &r.lottery, &r.createdAt); err != nil {
			fmt.Println("scan:", err)
			os.Exit(1)
		}
		list = append(list, r)
	}

	fmt.Printf("孤儿方案定义: %d 条\n", len(list))
	for _, r := range list {
		fmt.Printf("  %s  member=%s  name=%q  kind=%s  lottery=%s  created=%v\n",
			r.id, r.account, r.name, r.kind, r.lottery, r.createdAt)
	}

	if !*apply {
		if len(list) > 0 {
			fmt.Println("\n预览模式，未删除。执行: go run ./cmd/purge-orphan-schemes -apply")
		}
		return
	}

	tag, err := pool.Exec(ctx, `
DELETE FROM scheme_definitions d
WHERE NOT EXISTS (SELECT 1 FROM scheme_instances i WHERE i.definition_id = d.id)`)
	if err != nil {
		fmt.Println("delete:", err)
		os.Exit(1)
	}
	fmt.Printf("\n已删除 %d 条孤儿方案定义\n", tag.RowsAffected())
}
