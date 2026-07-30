.PHONY: run build test tidy migrate-up migrate-down migrate-status sqlc guaji-smoke guaji-capture guaji-audit-ws-keys scheme-e2e

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

diag-lookback:
	go run ./cmd/diag-lookback-schema/

lookback-schema-fix:
	go run ./cmd/diag-lookback-schema/ -fix

diag-simbet:
	go run ./cmd/diag-simbet-schema/

simbet-schema-fix:
	go run ./cmd/diag-simbet-schema/ -fix

guaji-smoke:
	go run ./cmd/guaji-smoke

guaji-capture:
	go run ./cmd/guaji-capture

# 对照 guaji_ws_key 与 v6hs1 WS/REST 期号；CI 加 -ci（需 GUAJI_ENABLED + DB 已跑 00123）
guaji-audit-ws-keys:
	go run ./cmd/audit-ws-keys -ci

guaji-audit-ws-keys-report:
	go run ./cmd/audit-ws-keys

# 只读对账：号池闭包 / 注数 / 期号归属 / 判定 / 资金守恒 / 冷热可达性
# 方案全链路 E2E：建方案 → 启动 → 出号下注 → 结算 → 查询，覆盖全部 7 种运行类型。
# 走真实库、全程模拟盘、跑完自清理。建议先停掉本地后端，避免它的 worker 并发推同一批实例。
scheme-e2e:
	go test ./internal/schemes/ -run "E2E$$|LifecycleE2E" -v -count=1

scheme-audit:
	go run ./cmd/scheme-audit

scheme-audit-ci:
	go run ./cmd/scheme-audit -days 2 -ci

guaji-catalog-sync:
	go run ./cmd/guaji-catalog-sync

guaji-rules-sync:
	go run ./cmd/guaji-rules-sync
