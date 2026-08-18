package cloudrealtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestReconcilerOnlyLeaderScansAndAdvancesCompositeCursor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	initial := now.Add(-5 * time.Minute)
	sameTimestamp := now.Add(-2 * time.Minute)
	follower := &fakeReconcileSession{}
	leader := &fakeReconcileSession{leader: true}
	leader.query = func(call int, after time.Time, afterID string, limit int) ([]sqlcdb.SchemeRealtimeChange, error) {
		switch call {
		case 1:
			assertReconcileQuery(t, after, afterID, limit, initial, "", 2)
			return []sqlcdb.SchemeRealtimeChange{
				{MemberID: 7, InstanceID: "a", UpdatedAt: sameTimestamp},
				{MemberID: 7, InstanceID: "b", UpdatedAt: sameTimestamp},
			}, nil
		case 2:
			assertReconcileQuery(t, after, afterID, limit, sameTimestamp, "b", 2)
			return nil, nil
		default:
			t.Fatalf("unexpected query call %d", call)
			return nil, nil
		}
	}
	pool := &fakeReconcileAcquirer{sessions: []*fakeReconcileSession{follower, leader}}
	marker := &recordingReconcileMarker{}
	reconciler := newReconciler(pool, marker, ReconcilerConfig{
		Interval: time.Second,
		Batch:    2,
		now:      func() time.Time { return now },
	})

	reconciler.reconcile(context.Background())
	if follower.queryCalls != 0 || !follower.released {
		t.Fatalf("follower queryCalls=%d released=%v, want 0 and true", follower.queryCalls, follower.released)
	}
	if diagnostics := reconciler.Diagnostics(); diagnostics.Leader || !diagnostics.CursorUpdatedAt.Equal(initial) || diagnostics.CursorID != "" {
		t.Fatalf("follower diagnostics=%+v", diagnostics)
	}

	reconciler.reconcile(context.Background())
	if got := marker.snapshot(); fmt.Sprint(got) != fmt.Sprint([]reconcileMark{{7, "a"}, {7, "b"}}) {
		t.Fatalf("marks=%v", got)
	}
	if diagnostics := reconciler.Diagnostics(); !diagnostics.Leader || !diagnostics.CursorUpdatedAt.Equal(sameTimestamp) || diagnostics.CursorID != "b" {
		t.Fatalf("leader diagnostics=%+v", diagnostics)
	}
}

func TestReconcilerMarksEveryRowBeforeAdvancingCursor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	initial := now.Add(-5 * time.Minute)
	firstAt := now.Add(-4 * time.Minute)
	secondAt := now.Add(-3 * time.Minute)
	session := &fakeReconcileSession{leader: true}
	session.query = func(call int, after time.Time, afterID string, limit int) ([]sqlcdb.SchemeRealtimeChange, error) {
		if call != 1 {
			t.Fatalf("unexpected query call %d", call)
		}
		return []sqlcdb.SchemeRealtimeChange{
			{MemberID: 7, InstanceID: "a", UpdatedAt: firstAt},
			{MemberID: 8, InstanceID: "b", UpdatedAt: secondAt},
		}, nil
	}
	pool := &fakeReconcileAcquirer{sessions: []*fakeReconcileSession{session}}
	var reconciler *Reconciler
	var cursors []reconcileCursor
	marker := reconcileMarkerFunc(func(int64, string) {
		diagnostics := reconciler.Diagnostics()
		cursors = append(cursors, reconcileCursor{updatedAt: diagnostics.CursorUpdatedAt, id: diagnostics.CursorID})
	})
	reconciler = newReconciler(pool, marker, ReconcilerConfig{
		Interval: time.Second,
		Batch:    10,
		now:      func() time.Time { return now },
	})

	reconciler.reconcile(context.Background())

	want := []reconcileCursor{{updatedAt: initial}, {updatedAt: firstAt, id: "a"}}
	if fmt.Sprint(cursors) != fmt.Sprint(want) {
		t.Fatalf("cursor observed by marker=%v want=%v", cursors, want)
	}
	if diagnostics := reconciler.Diagnostics(); !diagnostics.CursorUpdatedAt.Equal(secondAt) || diagnostics.CursorID != "b" {
		t.Fatalf("final diagnostics=%+v", diagnostics)
	}
}

func TestReconcilerRetriesSameCursorAfterQueryFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	initial := now.Add(-5 * time.Minute)
	queryFailure := errors.New("query failed")
	var cursors []reconcileCursor
	failed := &fakeReconcileSession{leader: true}
	failed.query = func(_ int, after time.Time, afterID string, _ int) ([]sqlcdb.SchemeRealtimeChange, error) {
		cursors = append(cursors, reconcileCursor{updatedAt: after, id: afterID})
		return nil, queryFailure
	}
	retried := &fakeReconcileSession{leader: true}
	retried.query = func(_ int, after time.Time, afterID string, _ int) ([]sqlcdb.SchemeRealtimeChange, error) {
		cursors = append(cursors, reconcileCursor{updatedAt: after, id: afterID})
		return nil, nil
	}
	reconciler := newReconciler(
		&fakeReconcileAcquirer{sessions: []*fakeReconcileSession{failed, retried}},
		reconcileMarkerFunc(func(int64, string) {}),
		ReconcilerConfig{Interval: time.Second, Batch: 10, now: func() time.Time { return now }},
	)

	reconciler.reconcile(context.Background())
	if !failed.released {
		t.Fatal("failed scan did not release its connection")
	}
	if diagnostics := reconciler.Diagnostics(); diagnostics.Leader || diagnostics.Errors != 1 || diagnostics.LastError != queryFailure.Error() || !diagnostics.CursorUpdatedAt.Equal(initial) || diagnostics.CursorID != "" {
		t.Fatalf("failure diagnostics=%+v", diagnostics)
	}

	reconciler.reconcile(context.Background())
	want := []reconcileCursor{{updatedAt: initial}, {updatedAt: initial}}
	if fmt.Sprint(cursors) != fmt.Sprint(want) {
		t.Fatalf("query cursors=%v want=%v", cursors, want)
	}
}

func TestReconcilerYieldsAfterConfiguredBatchBudget(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	session := &fakeReconcileSession{leader: true}
	session.query = func(call int, _ time.Time, _ string, limit int) ([]sqlcdb.SchemeRealtimeChange, error) {
		if limit != 1 {
			t.Fatalf("limit=%d want=1", limit)
		}
		return []sqlcdb.SchemeRealtimeChange{{
			MemberID:   int64(call),
			InstanceID: fmt.Sprintf("instance-%d", call),
			UpdatedAt:  now.Add(time.Duration(call) * time.Second),
		}}, nil
	}
	marker := &recordingReconcileMarker{}
	reconciler := newReconciler(
		&fakeReconcileAcquirer{sessions: []*fakeReconcileSession{session}},
		marker,
		ReconcilerConfig{Interval: time.Second, Batch: 1, now: func() time.Time { return now }},
	)

	reconciler.reconcile(context.Background())
	if session.queryCalls != 4 || len(marker.snapshot()) != 4 {
		t.Fatalf("first tick queryCalls=%d marks=%d want=4", session.queryCalls, len(marker.snapshot()))
	}
	if diagnostics := reconciler.Diagnostics(); diagnostics.CursorID != "instance-4" {
		t.Fatalf("first tick diagnostics=%+v", diagnostics)
	}

	reconciler.reconcile(context.Background())
	if session.queryCalls != 8 || len(marker.snapshot()) != 8 {
		t.Fatalf("second tick queryCalls=%d marks=%d want=8", session.queryCalls, len(marker.snapshot()))
	}
}

type reconcileCursor struct {
	updatedAt time.Time
	id        string
}

type fakeReconcileAcquirer struct {
	mu       sync.Mutex
	sessions []*fakeReconcileSession
	err      error
}

func (a *fakeReconcileAcquirer) Acquire(context.Context) (reconcileSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return nil, a.err
	}
	if len(a.sessions) == 0 {
		return nil, errors.New("no fake reconcile session available")
	}
	session := a.sessions[0]
	a.sessions = a.sessions[1:]
	return session, nil
}

type fakeReconcileSession struct {
	mu         sync.Mutex
	leader     bool
	lockErr    error
	query      func(call int, after time.Time, afterID string, limit int) ([]sqlcdb.SchemeRealtimeChange, error)
	queryCalls int
	released   bool
}

func (s *fakeReconcileSession) TryAdvisoryLock(context.Context, int64) (bool, error) {
	return s.leader, s.lockErr
}

func (s *fakeReconcileSession) ListSchemeRealtimeChanges(_ context.Context, after time.Time, afterID string, limit int) ([]sqlcdb.SchemeRealtimeChange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queryCalls++
	if s.query == nil {
		return nil, errors.New("unexpected query")
	}
	return s.query(s.queryCalls, after, afterID, limit)
}

func (s *fakeReconcileSession) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.released = true
}

type reconcileMark struct {
	memberID   int64
	instanceID string
}

type recordingReconcileMarker struct {
	mu    sync.Mutex
	marks []reconcileMark
}

func (m *recordingReconcileMarker) MarkScheme(memberID int64, instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.marks = append(m.marks, reconcileMark{memberID: memberID, instanceID: instanceID})
}

func (m *recordingReconcileMarker) snapshot() []reconcileMark {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]reconcileMark(nil), m.marks...)
}

type reconcileMarkerFunc func(memberID int64, instanceID string)

func (f reconcileMarkerFunc) MarkScheme(memberID int64, instanceID string) {
	f(memberID, instanceID)
}

func assertReconcileQuery(t *testing.T, gotAt time.Time, gotID string, gotLimit int, wantAt time.Time, wantID string, wantLimit int) {
	t.Helper()
	if !gotAt.Equal(wantAt) || gotID != wantID || gotLimit != wantLimit {
		t.Fatalf("query cursor=(%s,%q) limit=%d want=(%s,%q) limit=%d", gotAt, gotID, gotLimit, wantAt, wantID, wantLimit)
	}
}
