.PHONY: run build test tidy migrate-up migrate-down migrate-status sqlc guaji-smoke guaji-capture guaji-audit-ws-keys

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

guaji-catalog-sync:
	go run ./cmd/guaji-catalog-sync

guaji-rules-sync:
	go run ./cmd/guaji-rules-sync
