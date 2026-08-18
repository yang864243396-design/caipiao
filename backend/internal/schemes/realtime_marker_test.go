package schemes

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

type recordingMarker struct {
	mu   sync.Mutex
	refs []RealtimeInstanceRef
}

func (m *recordingMarker) MarkScheme(memberID int64, instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID})
}

func TestWorkerNotifySchemeInstanceMarksRealtime(t *testing.T) {
	marker := &recordingMarker{}
	w := &Worker{realtime: marker}
	w.notifySchemeInstance(context.Background(), 7, "inst-a", "real", "running", "cloud_active")

	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if !reflect.DeepEqual(marker.refs, want) {
		t.Fatalf("refs=%v want=%v", marker.refs, want)
	}
}

func TestServiceRealtimeMarkerIgnoresInvalidReferences(t *testing.T) {
	marker := &recordingMarker{}
	s := &Service{}
	s.SetRealtimeMarker(marker)

	s.markScheme(0, "inst-a")
	s.markScheme(7, "")
	s.markScheme(7, "   ")

	marker.mu.Lock()
	defer marker.mu.Unlock()
	if len(marker.refs) != 0 {
		t.Fatalf("refs=%v want no invalid marks", marker.refs)
	}
}
