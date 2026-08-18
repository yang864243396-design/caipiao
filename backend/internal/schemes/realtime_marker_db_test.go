package schemes_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/schemes"
)

type recordingDBMarker struct {
	mu     sync.Mutex
	refs   []schemes.RealtimeInstanceRef
	onMark func(memberID int64, instanceID string)
}

func (m *recordingDBMarker) MarkScheme(memberID int64, instanceID string) {
	if m.onMark != nil {
		m.onMark(memberID, instanceID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, schemes.RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID})
}

func (m *recordingDBMarker) reset(onMark func(memberID int64, instanceID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = nil
	m.onMark = onMark
}

func (m *recordingDBMarker) assertRefs(t *testing.T, want []schemes.RealtimeInstanceRef) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if !reflect.DeepEqual(m.refs, want) {
		t.Fatalf("refs=%v want=%v", m.refs, want)
	}
}

func createRealtimeDefinition(t *testing.T, env *e2eEnv, suffix string) schemes.Definition {
	t.Helper()
	def, err := env.svc.CreateDefinition(context.Background(), env.account, schemes.CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  fmt.Sprintf("realtime-%s-%d", suffix, time.Now().UnixNano()),
		LotteryCode: env.lottery,
		RunTypeID:   "fixed_number",
		PlayTypeID:  env.playTypeID,
		SubPlayID:   env.subPlayID,
	})
	if err != nil {
		t.Fatalf("CreateDefinition: %v", err)
	}
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE id = $1`, def.ID)
	})
	return def
}

func TestCommittedDefinitionMutationsMarkRealtime(t *testing.T) {
	t.Run("add definition to cloud", func(t *testing.T) {
		env := newE2EEnv(t)
		marker := &recordingDBMarker{}
		env.svc.SetRealtimeMarker(marker)
		def := createRealtimeDefinition(t, env, "add")

		marker.reset(func(memberID int64, instanceID string) {
			var gotMemberID int64
			if err := env.pool.QueryRow(context.Background(),
				`SELECT member_id FROM scheme_instances WHERE id = $1`, instanceID).Scan(&gotMemberID); err != nil {
				t.Errorf("marker ran before committed instance was visible: %v", err)
			}
			if gotMemberID != memberID {
				t.Errorf("committed member id=%d marker member id=%d", gotMemberID, memberID)
			}
		})
		result, err := env.svc.AddDefinitionToCloud(context.Background(), env.account, def.ID, "private", schemes.AddToCloudConfigPatch{
			SchemeFunds: "1000",
			BetUnit:     "1",
		})
		if err != nil {
			t.Fatalf("AddDefinitionToCloud: %v", err)
		}
		marker.assertRefs(t, []schemes.RealtimeInstanceRef{{MemberID: env.memberID, InstanceID: result.Instance.ID}})
	})

	t.Run("definition edit synchronizes instance", func(t *testing.T) {
		env := newE2EEnv(t)
		marker := &recordingDBMarker{}
		env.svc.SetRealtimeMarker(marker)
		def := createRealtimeDefinition(t, env, "update")
		result, err := env.svc.AddDefinitionToCloud(context.Background(), env.account, def.ID, "private", schemes.AddToCloudConfigPatch{
			SchemeFunds: "1000",
			BetUnit:     "1",
		})
		if err != nil {
			t.Fatalf("AddDefinitionToCloud: %v", err)
		}

		marker.reset(func(memberID int64, instanceID string) {
			var multiplier float64
			if err := env.pool.QueryRow(context.Background(),
				`SELECT multiplier::float8 FROM scheme_instances WHERE member_id = $1 AND id = $2`, memberID, instanceID).
				Scan(&multiplier); err != nil {
				t.Errorf("marker ran before committed multiplier was visible: %v", err)
			}
			if multiplier != 4 {
				t.Errorf("committed multiplier=%v want 4", multiplier)
			}
		})
		if _, err := env.svc.UpdateDefinition(context.Background(), env.account, def.ID, schemes.UpdateDefinitionPatch{
			HasMultCoeff: true,
			MultCoeff:    "4",
		}); err != nil {
			t.Fatalf("UpdateDefinition: %v", err)
		}
		marker.assertRefs(t, []schemes.RealtimeInstanceRef{{MemberID: env.memberID, InstanceID: result.Instance.ID}})
	})

	t.Run("share add to cloud", func(t *testing.T) {
		env := newE2EEnv(t)
		marker := &recordingDBMarker{}
		env.svc.SetRealtimeMarker(marker)
		def := createRealtimeDefinition(t, env, "share")
		donor, err := env.svc.AddDefinitionToCloud(context.Background(), env.account, def.ID, "public", schemes.AddToCloudConfigPatch{
			SchemeFunds: "1000",
			BetUnit:     "1",
		})
		if err != nil {
			t.Fatalf("AddDefinitionToCloud(public): %v", err)
		}
		if donor.ShareSnapshotID == "" {
			t.Fatal("public add-to-cloud returned blank share snapshot id")
		}

		marker.reset(func(memberID int64, instanceID string) {
			var gotMemberID int64
			if err := env.pool.QueryRow(context.Background(),
				`SELECT member_id FROM scheme_instances WHERE id = $1`, instanceID).Scan(&gotMemberID); err != nil {
				t.Errorf("marker ran before committed shared instance was visible: %v", err)
			}
			if gotMemberID != memberID {
				t.Errorf("committed member id=%d marker member id=%d", gotMemberID, memberID)
			}
		})
		follow, err := env.svc.ShareAddToCloud(context.Background(), env.account, donor.ShareSnapshotID, schemes.ShareAddToCloudInput{})
		if err != nil {
			t.Fatalf("ShareAddToCloud: %v", err)
		}
		marker.assertRefs(t, []schemes.RealtimeInstanceRef{{MemberID: env.memberID, InstanceID: follow.Instance.ID}})
	})
}

func TestDeleteDefinitionMarksRemovedInstance(t *testing.T) {
	env := newE2EEnv(t)
	marker := &recordingDBMarker{}
	env.svc.SetRealtimeMarker(marker)
	def := createRealtimeDefinition(t, env, "delete")
	result, err := env.svc.AddDefinitionToCloud(context.Background(), env.account, def.ID, "private", schemes.AddToCloudConfigPatch{
		SchemeFunds: "1000",
		BetUnit:     "1",
	})
	if err != nil {
		t.Fatalf("AddDefinitionToCloud: %v", err)
	}
	originalRef := schemes.RealtimeInstanceRef{MemberID: result.Instance.MemberID, InstanceID: result.Instance.ID}

	marker.reset(func(memberID int64, instanceID string) {
		var id string
		err := env.pool.QueryRow(context.Background(),
			`SELECT id FROM scheme_instances WHERE member_id = $1 AND id = $2`, memberID, instanceID).Scan(&id)
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("marker ran before cascade deletion committed: err=%v id=%q", err, id)
		}
	})
	if err := env.svc.DeleteDefinition(context.Background(), env.account, def.ID); err != nil {
		t.Fatalf("DeleteDefinition: %v", err)
	}
	marker.assertRefs(t, []schemes.RealtimeInstanceRef{originalRef})
}
