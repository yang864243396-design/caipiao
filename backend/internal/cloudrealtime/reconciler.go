package cloudrealtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeevents"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultReconcileInterval = 5 * time.Second
	defaultReconcileBatch    = 500
	maxFullBatchesPerTick    = 4
	reconcileCursorLookback  = 5 * time.Minute
	reconcileUnlockTimeout   = 2 * time.Second

	// "CloudRec" encoded as a positive, stable signed 64-bit advisory key.
	reconcileAdvisoryLockKey int64 = 0x436c6f7564526563
)

var (
	errReconcileAdvisoryUnlockFailed       = errors.New("cloud realtime reconciler advisory unlock failed")
	errReconcileAdvisoryUnlockNotConfirmed = errors.New("cloud realtime reconciler advisory unlock not confirmed")
	errReconcilePhysicalCloseFailed        = errors.New("cloud realtime reconciler physical connection close failed")
)

type ReconcilerConfig struct {
	Interval  time.Duration
	Batch     int
	now       func() time.Time
	newTicker func(time.Duration) (<-chan time.Time, func())
}

type ReconcilerDiagnostics struct {
	Leader          bool          `json:"leader"`
	CursorUpdatedAt time.Time     `json:"cursorUpdatedAt"`
	CursorID        string        `json:"cursorId"`
	CursorLag       time.Duration `json:"cursorLag"`
	Scans           uint64        `json:"scans"`
	RowsMarked      uint64        `json:"rowsMarked"`
	Errors          uint64        `json:"errors"`
	LastScanAt      time.Time     `json:"lastScanAt,omitempty"`
	LastSuccessAt   time.Time     `json:"lastSuccessAt,omitempty"`
	LastError       string        `json:"lastError,omitempty"`
}

type Reconciler struct {
	acquirer reconcileAcquirer
	marker   schemeevents.Marker
	cfg      ReconcilerConfig

	mu          sync.Mutex
	diagnostics ReconcilerDiagnostics
	session     reconcileSession
}

type reconcileAcquirer interface {
	Acquire(context.Context) (reconcileSession, error)
}

type reconcileSession interface {
	TryAdvisoryLock(context.Context, int64) (bool, error)
	ListSchemeRealtimeChanges(context.Context, time.Time, string, int) ([]sqlcdb.SchemeRealtimeChange, error)
	Release() error
}

func NewReconciler(pool *pgxpool.Pool, marker schemeevents.Marker, cfg ReconcilerConfig) *Reconciler {
	return newReconciler(pgxReconcileAcquirer{pool: pool}, marker, cfg)
}

func newReconciler(acquirer reconcileAcquirer, marker schemeevents.Marker, cfg ReconcilerConfig) *Reconciler {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultReconcileInterval
	}
	if cfg.Batch <= 0 {
		cfg.Batch = defaultReconcileBatch
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}
	if cfg.newTicker == nil {
		cfg.newTicker = newReconcileTicker
	}
	initialCursor := cfg.now().UTC().Add(-reconcileCursorLookback)
	return &Reconciler{
		acquirer: acquirer,
		marker:   marker,
		cfg:      cfg,
		diagnostics: ReconcilerDiagnostics{
			CursorUpdatedAt: initialCursor,
		},
	}
}

func (r *Reconciler) Run(ctx context.Context) {
	ticks, stop := r.cfg.newTicker(r.cfg.Interval)
	defer stop()
	defer r.releaseSession()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			r.reconcile(ctx)
		}
	}
}

func (r *Reconciler) Diagnostics() ReconcilerDiagnostics {
	r.mu.Lock()
	defer r.mu.Unlock()
	diagnostics := r.diagnostics
	diagnostics.CursorLag = r.cfg.now().UTC().Sub(diagnostics.CursorUpdatedAt)
	if diagnostics.CursorLag < 0 {
		diagnostics.CursorLag = 0
	}
	return diagnostics
}

func (r *Reconciler) reconcile(ctx context.Context) {
	r.recordScan()
	if r.session == nil && !r.acquireLeadership(ctx) {
		return
	}

	for batch := 0; batch < maxFullBatchesPerTick; batch++ {
		cursorAt, cursorID := r.cursor()
		changes, err := r.session.ListSchemeRealtimeChanges(ctx, cursorAt, cursorID, r.cfg.Batch)
		if err != nil {
			r.recordError(err)
			r.releaseSession()
			return
		}
		for _, change := range changes {
			r.marker.MarkScheme(change.MemberID, change.InstanceID)
			r.advanceCursor(change)
		}
		if len(changes) < r.cfg.Batch {
			r.recordSuccess()
			return
		}
	}
	r.recordSuccess()
}

func (r *Reconciler) acquireLeadership(ctx context.Context) bool {
	session, err := r.acquirer.Acquire(ctx)
	if err != nil {
		r.recordError(err)
		return false
	}
	leader, err := session.TryAdvisoryLock(ctx, reconcileAdvisoryLockKey)
	if err != nil {
		r.recordError(err)
		r.releaseAcquiredSession(session)
		return false
	}
	if !leader {
		r.releaseAcquiredSession(session)
		r.setLeader(false)
		return false
	}
	r.session = session
	r.setLeader(true)
	return true
}

func (r *Reconciler) releaseSession() {
	if r.session != nil {
		r.releaseAcquiredSession(r.session)
		r.session = nil
	}
	r.setLeader(false)
}

func (r *Reconciler) releaseAcquiredSession(session reconcileSession) {
	if err := session.Release(); err != nil {
		r.recordError(err)
	}
}

func (r *Reconciler) cursor() (time.Time, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.diagnostics.CursorUpdatedAt, r.diagnostics.CursorID
}

func (r *Reconciler) advanceCursor(change sqlcdb.SchemeRealtimeChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics.CursorUpdatedAt = change.UpdatedAt.UTC()
	r.diagnostics.CursorID = change.InstanceID
	r.diagnostics.RowsMarked++
}

func (r *Reconciler) recordScan() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics.Scans++
	r.diagnostics.LastScanAt = r.cfg.now().UTC()
}

func (r *Reconciler) recordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics.LastSuccessAt = r.cfg.now().UTC()
}

func (r *Reconciler) recordError(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics.Errors++
	r.diagnostics.LastError = err.Error()
}

func (r *Reconciler) setLeader(leader bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.diagnostics.Leader = leader
}

func newReconcileTicker(interval time.Duration) (<-chan time.Time, func()) {
	ticker := time.NewTicker(interval)
	return ticker.C, ticker.Stop
}

type pgxReconcileAcquirer struct {
	pool *pgxpool.Pool
}

func (a pgxReconcileAcquirer) Acquire(ctx context.Context) (reconcileSession, error) {
	if a.pool == nil {
		return nil, errors.New("cloud realtime reconciler database pool is nil")
	}
	conn, err := a.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxReconcileSession{
		conn:        &pgxPooledReconcileConn{conn: conn},
		listChanges: sqlcdb.New(conn).ListSchemeRealtimeChanges,
	}, nil
}

type pgxReconcileSession struct {
	conn            reconcileConn
	listChanges     func(context.Context, time.Time, string, int) ([]sqlcdb.SchemeRealtimeChange, error)
	cleanupRequired bool
	lockKey         int64
}

type reconcileConn interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Release()
	ClosePhysical(context.Context) error
}

func (s *pgxReconcileSession) TryAdvisoryLock(ctx context.Context, key int64) (bool, error) {
	s.cleanupRequired = true
	s.lockKey = key
	var acquired bool
	if err := s.conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, err
	}
	s.cleanupRequired = acquired
	return acquired, nil
}

func (s *pgxReconcileSession) ListSchemeRealtimeChanges(ctx context.Context, after time.Time, afterID string, limit int) ([]sqlcdb.SchemeRealtimeChange, error) {
	return s.listChanges(ctx, after, afterID, limit)
}

func (s *pgxReconcileSession) Release() error {
	if !s.cleanupRequired {
		s.conn.Release()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), reconcileUnlockTimeout)
	var unlocked bool
	err := s.conn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", s.lockKey).Scan(&unlocked)
	cancel()
	if err == nil && unlocked {
		s.cleanupRequired = false
		s.conn.Release()
		return nil
	}

	cleanupErr := errReconcileAdvisoryUnlockNotConfirmed
	if err != nil {
		cleanupErr = errReconcileAdvisoryUnlockFailed
	}
	ctx, cancel = context.WithTimeout(context.Background(), reconcileUnlockTimeout)
	closeErr := s.conn.ClosePhysical(ctx)
	cancel()
	if closeErr != nil {
		return errors.Join(cleanupErr, errReconcilePhysicalCloseFailed)
	}
	return cleanupErr
}

type pgxPooledReconcileConn struct {
	conn *pgxpool.Conn
}

func (c *pgxPooledReconcileConn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.conn.QueryRow(ctx, sql, args...)
}

func (c *pgxPooledReconcileConn) Release() {
	c.conn.Release()
}

func (c *pgxPooledReconcileConn) ClosePhysical(ctx context.Context) error {
	return c.conn.Hijack().Close(ctx)
}
