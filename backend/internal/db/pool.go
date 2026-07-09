package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string, maxConns, minConns int) (*Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL 未配置（或 DB_HOST 为空）")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	if maxConns <= 0 {
		maxConns = 25
	}
	if minConns < 0 {
		minConns = 0
	}
	if int32(minConns) > int32(maxConns) {
		minConns = maxConns
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(minConns)
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

func (p *Pool) ServerVersion(ctx context.Context) (string, error) {
	var version string
	err := p.QueryRow(ctx, "SHOW server_version").Scan(&version)
	return version, err
}
