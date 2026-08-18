package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/middleware"
	"caipiao/backend/internal/schemes"
)

type handlerRecordingMarker struct {
	mu     sync.Mutex
	refs   []schemes.RealtimeInstanceRef
	onMark func(memberID int64, instanceID string)
}

func (m *handlerRecordingMarker) MarkScheme(memberID int64, instanceID string) {
	if m.onMark != nil {
		m.onMark(memberID, instanceID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, schemes.RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID})
}

func TestCloudActionPublisherMarksCommittedInstance(t *testing.T) {
	env := newHandlerEnv(t)
	if _, err := env.pool.Exec(context.Background(),
		`UPDATE scheme_instances SET status = 'running', status_reason = '' WHERE id = $1`, env.schemeID); err != nil {
		t.Fatalf("seed running instance: %v", err)
	}

	marker := &handlerRecordingMarker{}
	marker.onMark = func(memberID int64, instanceID string) {
		var status string
		if err := env.pool.QueryRow(context.Background(),
			`SELECT status FROM scheme_instances WHERE member_id = $1 AND id = $2`, memberID, instanceID).Scan(&status); err != nil {
			t.Errorf("marker ran before committed action was visible: %v", err)
		}
		if status != "pending" {
			t.Errorf("status at marker=%q want pending", status)
		}
	}
	env.h.schemes = schemes.NewService(env.pool, nil)
	env.h.SetRealtimeMarker(marker)

	req := httptest.NewRequest(http.MethodPost, "/client/cloud/instances/stop", nil)
	req.SetPathValue("instanceId", env.schemeID)
	req = req.WithContext(middleware.WithClaims(req.Context(),
		auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: env.account}}))
	rec := httptest.NewRecorder()
	env.h.CloudInstanceStop(rec, req)
	code, message, _ := decodeEnvelope(t, rec)
	if code != apix.CodeOK {
		t.Fatalf("response code=%d message=%q", code, message)
	}

	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []schemes.RealtimeInstanceRef{{MemberID: env.memberID, InstanceID: env.schemeID}}
	if !reflect.DeepEqual(marker.refs, want) {
		t.Fatalf("refs=%v want=%v", marker.refs, want)
	}
}
