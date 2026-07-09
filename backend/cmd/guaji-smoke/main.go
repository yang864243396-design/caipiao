package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := guaji.LoadConfigFromEnv()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if !cfg.Enabled {
		slog.Error("GUAJI_ENABLED 未开启；请在 backend/.env 设置 GUAJI_ENABLED=true")
		os.Exit(1)
	}
	if err := cfg.Valid(); err != nil {
		slog.Error("guaji 配置无效", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client := guaji.NewClient(cfg)
	result := client.Probe(ctx)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)

	switch {
	case !result.HTTPReachable:
		slog.Error("HTTP 不可达", "err", result.HTTPError)
		os.Exit(2)
	case !result.WSReachable:
		slog.Warn("WS 不可达（部分网络环境正常；开奖 T3 前需确认）", "err", result.WSError)
	case cfg.TestUsername != "" && !result.LoginOK && !result.MFARequired:
		slog.Error("测试账号登录失败", "err", result.LoginError)
		os.Exit(3)
	case result.MFARequired:
		slog.Warn("测试账号需 MFA；WS 可达，绑号流程 T1 处理")
	default:
		bal := 0.0
		if result.BalanceCNY != nil {
			bal = *result.BalanceCNY
		}
		slog.Info("guaji smoke ok",
			"http", result.HTTPReachable,
			"ws", result.WSReachable,
			"login", result.LoginOK,
			"balanceCny", bal,
		)
	}
}
