package schemes

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

const (
	maxSimSchemeConcurrent  = 5
	maxSimSchemeDailyStarts = 5
)

var (
	// ErrSimSchemeConcurrentLimit 同时运行的模拟方案已达上限。
	ErrSimSchemeConcurrentLimit = errors.New("最多可同时开启5个模拟测试方案，如需开启新方案，请先关闭一个已开启的方案")
	// ErrSimSchemeDailyStartLimit 当日模拟投注启动次数已达上限。
	ErrSimSchemeDailyStartLimit = errors.New("今天模拟投注运行次数已达上限")
	// ErrSimSchemeQuotaSchema 成员表缺少模拟配额列（未跑迁移 00130）。
	ErrSimSchemeQuotaSchema = errors.New("模拟投注配额未就绪，请在服务器执行数据库迁移后重试")
)

func isUndefinedColumnErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42703"
}

func mapSimQuotaSchemaErr(err error) error {
	if err == nil {
		return nil
	}
	if isUndefinedColumnErr(err) {
		slog.Error("sim scheme quota schema missing (need migrate 00130)", "err", err)
		return ErrSimSchemeQuotaSchema
	}
	return err
}

var (
	simQuotaSchemaMu sync.Mutex
	simQuotaSchemaOK bool
)

// ensureSimSchemeQuotaSchema 补齐 members 模拟配额列（等同迁移 00130，IF NOT EXISTS 可重复执行）。
// 线上若只发版未 migrate，开启模拟方案时自动自愈，避免长期无法开启。
func (s *Service) ensureSimSchemeQuotaSchema(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return ErrUnavailable
	}
	simQuotaSchemaMu.Lock()
	defer simQuotaSchemaMu.Unlock()
	if simQuotaSchemaOK {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
ALTER TABLE members
    ADD COLUMN IF NOT EXISTS sim_scheme_starts_date DATE,
    ADD COLUMN IF NOT EXISTS sim_scheme_starts_count INT NOT NULL DEFAULT 0`)
	if err != nil {
		slog.Error("auto-apply sim scheme quota columns failed", "err", err)
		return err
	}
	simQuotaSchemaOK = true
	slog.Info("ensured members sim scheme quota columns (00130)")
	return nil
}

type SimSchemeQuota struct {
	TodayStarts      int `json:"todayStarts"`
	TodayStartsLimit int `json:"todayStartsLimit"`
	Running          int `json:"running"`
	RunningLimit     int `json:"runningLimit"`
}

func shanghaiTodayDate(now time.Time) time.Time {
	loc := timeutil.PlatformLocation()
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func (s *Service) countRunningSimSchemes(ctx context.Context, memberID int64) (int, error) {
	if s == nil || s.pool == nil {
		return 0, ErrUnavailable
	}
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM scheme_instances
WHERE member_id = $1
  AND sim_bet = true
  AND status = 'running'
`, memberID).Scan(&n)
	return n, mapSimQuotaSchemaErr(err)
}

func (s *Service) readSimTodayStarts(ctx context.Context, memberID int64, today time.Time) (int, error) {
	if s == nil || s.pool == nil {
		return 0, ErrUnavailable
	}
	var date pgtype.Date
	var count int32
	err := s.pool.QueryRow(ctx, `
SELECT sim_scheme_starts_date, sim_scheme_starts_count
FROM members
WHERE id = $1
`, memberID).Scan(&date, &count)
	if err != nil {
		return 0, mapSimQuotaSchemaErr(err)
	}
	if !date.Valid {
		return 0, nil
	}
	d := date.Time.In(timeutil.PlatformLocation())
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, timeutil.PlatformLocation())
	if !day.Equal(today) {
		return 0, nil
	}
	if count < 0 {
		return 0, nil
	}
	return int(count), nil
}

// reserveSimSchemeStart 在事务外预占一次当日启动额度；并发下用条件更新保证不超过上限。
func (s *Service) reserveSimSchemeStart(ctx context.Context, memberID int64, today time.Time) error {
	if s == nil || s.pool == nil {
		return ErrUnavailable
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE members
SET sim_scheme_starts_date = $2::date,
    sim_scheme_starts_count = CASE
      WHEN sim_scheme_starts_date IS DISTINCT FROM $2::date THEN 1
      ELSE sim_scheme_starts_count + 1
    END,
    updated_at = now()
WHERE id = $1
  AND (
    sim_scheme_starts_date IS DISTINCT FROM $2::date
    OR sim_scheme_starts_count < $3
  )
`, memberID, today.Format("2006-01-02"), maxSimSchemeDailyStarts)
	if err != nil {
		return mapSimQuotaSchemaErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSimSchemeDailyStartLimit
	}
	return nil
}

func (s *Service) releaseSimSchemeStart(ctx context.Context, memberID int64, today time.Time) {
	if s == nil || s.pool == nil {
		return
	}
	_, _ = s.pool.Exec(ctx, `
UPDATE members
SET sim_scheme_starts_count = GREATEST(sim_scheme_starts_count - 1, 0),
    updated_at = now()
WHERE id = $1
  AND sim_scheme_starts_date IS NOT DISTINCT FROM $2::date
  AND sim_scheme_starts_count > 0
`, memberID, today.Format("2006-01-02"))
}

func (s *Service) enforceSimSchemeStartQuota(ctx context.Context, memberID int64, now time.Time) (today time.Time, err error) {
	today = shanghaiTodayDate(now)
	// 先幂等补齐列，再计并发/日配额，避免只发版未 migrate 时无法开启模拟。
	if err := s.ensureSimSchemeQuotaSchema(ctx); err != nil {
		return today, ErrSimSchemeQuotaSchema
	}
	running, err := s.countRunningSimSchemes(ctx, memberID)
	if err != nil {
		return today, err
	}
	if running >= maxSimSchemeConcurrent {
		return today, ErrSimSchemeConcurrentLimit
	}
	if err := s.reserveSimSchemeStart(ctx, memberID, today); err != nil {
		return today, err
	}
	return today, nil
}

func (s *Service) simSchemeQuotaForMember(ctx context.Context, memberID int64) (SimSchemeQuota, error) {
	now := time.Now()
	today := shanghaiTodayDate(now)
	starts, err := s.readSimTodayStarts(ctx, memberID, today)
	if err != nil {
		return SimSchemeQuota{}, err
	}
	running, err := s.countRunningSimSchemes(ctx, memberID)
	if err != nil {
		return SimSchemeQuota{}, err
	}
	return SimSchemeQuota{
		TodayStarts:      starts,
		TodayStartsLimit: maxSimSchemeDailyStarts,
		Running:          running,
		RunningLimit:     maxSimSchemeConcurrent,
	}, nil
}

func (s *Service) GetSimSchemeQuota(ctx context.Context, account string) (SimSchemeQuota, error) {
	if s == nil || s.q == nil {
		return SimSchemeQuota{}, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SimSchemeQuota{}, member.ErrNotFound
		}
		return SimSchemeQuota{}, err
	}
	return s.simSchemeQuotaForMember(ctx, m.ID)
}
