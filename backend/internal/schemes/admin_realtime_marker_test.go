package schemes

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

type adminTransitionDB struct {
	currentStatus   string
	updateStatus    string
	updateErr       error
	updateCalls     int
	updateSucceeded bool
}

func (d *adminTransitionDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected exec")
}

func (d *adminTransitionDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (d *adminTransitionDB) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	if strings.Contains(query, "UPDATE scheme_instances") {
		d.updateCalls++
		return adminTransitionRow{scan: func(dest ...interface{}) error {
			if d.updateErr != nil {
				return d.updateErr
			}
			if err := scanAdminTransitionInstance(dest, "inst-admin", 77, d.updateStatus); err != nil {
				return err
			}
			d.updateSucceeded = true
			return nil
		}}
	}
	return adminTransitionRow{scan: func(dest ...interface{}) error {
		return scanAdminTransitionInstance(dest, "inst-admin", 77, d.currentStatus)
	}}
}

type adminTransitionRow struct {
	scan func(dest ...interface{}) error
}

func (r adminTransitionRow) Scan(dest ...interface{}) error {
	return r.scan(dest...)
}

func scanAdminTransitionInstance(dest []interface{}, id string, memberID int64, status string) error {
	values := []interface{}{
		id, "def-admin", memberID, "custom", "admin scheme", "g001", "lottery",
		status, "", pgtype.Numeric{}, pgtype.Numeric{}, int32(0), pgtype.Numeric{},
		pgtype.Numeric{}, pgtype.Numeric{}, int32(0), false, pgtype.Timestamptz{}, pgtype.Timestamptz{},
	}
	if len(dest) != len(values) {
		return errors.New("unexpected scan destination count")
	}
	for i, value := range values {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Ptr || target.IsNil() {
			return errors.New("invalid scan destination")
		}
		target.Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type orderedAdminMarker struct {
	db       *adminTransitionDB
	refs     []RealtimeInstanceRef
	tooEarly bool
}

func (m *orderedAdminMarker) MarkScheme(memberID int64, instanceID string) {
	if !m.db.updateSucceeded {
		m.tooEarly = true
	}
	m.refs = append(m.refs, RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID})
}

func TestAdminStatusTransitionsMarkRealtimeAfterSuccessfulUpdate(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		nextStatus    string
		transition    func(*Service) (Instance, error)
	}{
		{
			name:          "force stop",
			currentStatus: "running",
			nextStatus:    "soft_stopped",
			transition: func(s *Service) (Instance, error) {
				return s.AdminForceStop(context.Background(), "inst-admin")
			},
		},
		{
			name:          "release stop",
			currentStatus: "soft_stopped",
			nextStatus:    "paused",
			transition: func(s *Service) (Instance, error) {
				return s.AdminReleaseStop(context.Background(), "inst-admin")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &adminTransitionDB{currentStatus: tt.currentStatus, updateStatus: tt.nextStatus}
			marker := &orderedAdminMarker{db: db}
			svc := &Service{q: sqlcdb.New(db)}
			svc.SetRealtimeMarker(marker)

			got, err := tt.transition(svc)
			if err != nil {
				t.Fatalf("transition: %v", err)
			}
			if got.ID != "inst-admin" || got.MemberID != 77 || got.Status != tt.nextStatus {
				t.Fatalf("instance=%+v", got)
			}
			wantRefs := []RealtimeInstanceRef{{MemberID: 77, InstanceID: "inst-admin"}}
			if !reflect.DeepEqual(marker.refs, wantRefs) {
				t.Fatalf("marks=%v want=%v", marker.refs, wantRefs)
			}
			if marker.tooEarly {
				t.Fatal("realtime mark occurred before successful update scan")
			}
			if db.updateCalls != 1 {
				t.Fatalf("update calls=%d want=1", db.updateCalls)
			}
		})
	}
}

func TestAdminStatusTransitionFailuresDoNotMarkRealtime(t *testing.T) {
	t.Run("invalid action", func(t *testing.T) {
		db := &adminTransitionDB{currentStatus: "paused", updateStatus: "soft_stopped"}
		marker := &orderedAdminMarker{db: db}
		svc := &Service{q: sqlcdb.New(db)}
		svc.SetRealtimeMarker(marker)

		_, err := svc.AdminForceStop(context.Background(), "inst-admin")
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error=%v want ErrInvalidAdminAction", err)
		}
		if len(marker.refs) != 0 || db.updateCalls != 0 {
			t.Fatalf("marks=%v update calls=%d", marker.refs, db.updateCalls)
		}
	})

	t.Run("update query failure", func(t *testing.T) {
		queryErr := errors.New("update failed")
		db := &adminTransitionDB{
			currentStatus: "running",
			updateStatus:  "soft_stopped",
			updateErr:     queryErr,
		}
		marker := &orderedAdminMarker{db: db}
		svc := &Service{q: sqlcdb.New(db)}
		svc.SetRealtimeMarker(marker)

		_, err := svc.AdminForceStop(context.Background(), "inst-admin")
		if !errors.Is(err, queryErr) {
			t.Fatalf("error=%v want %v", err, queryErr)
		}
		if len(marker.refs) != 0 || db.updateSucceeded {
			t.Fatalf("marks=%v update succeeded=%v", marker.refs, db.updateSucceeded)
		}
	})
}
