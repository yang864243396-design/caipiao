package cloudrealtime

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/schemes"
)

func TestSubjectBuildersTrimPrefixAndRequireNumericMemberID(t *testing.T) {
	tests := []struct {
		name    string
		build   func(string, int64) (string, error)
		prefix  string
		member  int64
		want    string
		wantErr bool
	}{
		{name: "scheme", build: SchemeSubject, prefix: ".caipiao.", member: 7, want: "caipiao.client.7.scheme"},
		{name: "stats", build: StatsSubject, prefix: "..tenant.prod..", member: 42, want: "tenant.prod.client.42.cloud_stats"},
		{name: "empty prefix", build: SchemeSubject, prefix: "...", member: 7, wantErr: true},
		{name: "zero member", build: SchemeSubject, prefix: "caipiao", member: 0, wantErr: true},
		{name: "negative member", build: StatsSubject, prefix: "caipiao", member: -1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.build(tt.prefix, tt.member)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("subject=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestPublisherUsesExactDefaultCoalesceWindows(t *testing.T) {
	p := NewPublisher(&fakeSource{}, newRecordingBus(), Config{SubjectPrefix: "caipiao"})
	if p.cfg.SchemeCoalesce != 200*time.Millisecond {
		t.Fatalf("scheme coalesce=%s", p.cfg.SchemeCoalesce)
	}
	if p.cfg.StatsCoalesce != time.Second {
		t.Fatalf("stats coalesce=%s", p.cfg.StatsCoalesce)
	}
}

func TestPublisherCoalescesLatestSchemeMarksByMemberAndInstance(t *testing.T) {
	// Removing the dirty-key set or keying it by mark count makes this publish
	// duplicates and issue duplicate projection reads.
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{
		ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-a", UpdatedAt: "2026-08-18T00:00:01Z"}}},
	}}
	bus := realtimebus.NewMemory()
	defer bus.Close()
	payloads := subscribePayloads(t, bus, "caipiao.client.7.scheme")
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})

	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-a")
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}

	if calls := source.schemeCallCount(); calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	wantRefs := []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if refs := source.lastSchemeRefs(); !reflect.DeepEqual(refs, wantRefs) {
		t.Fatalf("refs=%v want=%v", refs, wantRefs)
	}
	message := receiveSchemeMessage(t, payloads)
	if message.SchemaVersion != 1 || len(message.Items) != 1 || message.Items[0].ID != "inst-a" {
		t.Fatalf("message=%+v", message)
	}
	if message.GeneratedAt == "" || message.RemovedIDs == nil {
		t.Fatalf("generatedAt=%q removedIds=%v", message.GeneratedAt, message.RemovedIDs)
	}
	assertNoPayload(t, payloads)
	diag := p.Diagnostics()
	if diag.AcceptedSchemeMarks != 3 || diag.CoalescedSchemeMarks != 2 || diag.SchemePublishes != 1 || diag.SchemeQueueSize != 0 {
		t.Fatalf("diagnostics=%+v", diag)
	}
}

func TestPublisherPublishesSeparateFullSnapshotsForMembers(t *testing.T) {
	// Publishing one combined payload or reusing the first member's subject
	// leaks account-scoped state and fails this test.
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{
		ItemsByMember: map[int64][]schemes.Instance{
			7: {{ID: "inst-7", UpdatedAt: "2026-08-18T00:00:01Z"}},
			8: {{ID: "inst-8", UpdatedAt: "2026-08-18T00:00:02Z"}},
		},
		RemovedByMember: map[int64][]string{8: {"gone-8"}},
	}}
	bus := realtimebus.NewMemory()
	defer bus.Close()
	member7 := subscribePayloads(t, bus, "caipiao.client.7.scheme")
	member8 := subscribePayloads(t, bus, "caipiao.client.8.scheme")
	p := NewPublisher(source, bus, Config{SubjectPrefix: ".caipiao."})
	p.MarkScheme(8, "inst-8")
	p.MarkScheme(8, "gone-8")
	p.MarkScheme(7, "inst-7")

	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	got7 := receiveSchemeMessage(t, member7)
	got8 := receiveSchemeMessage(t, member8)
	if len(got7.Items) != 1 || got7.Items[0].ID != "inst-7" || len(got7.RemovedIDs) != 0 {
		t.Fatalf("member 7 message=%+v", got7)
	}
	if len(got8.Items) != 1 || got8.Items[0].ID != "inst-8" || !reflect.DeepEqual(got8.RemovedIDs, []string{"gone-8"}) {
		t.Fatalf("member 8 message=%+v", got8)
	}
}

func TestPublisherSourceFailureRequeuesAffectedMemberWithoutBlockingOthers(t *testing.T) {
	// A single failed bulk load must fall back to isolated member loads. If the
	// whole batch is blindly restored, member 8 never publishes in this flush.
	source := &fakeSource{
		schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{
			7: {{ID: "inst-7"}},
			8: {{ID: "inst-8"}},
		}},
		schemeErrMembers: map[int64]bool{7: true},
	}
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkScheme(7, "inst-7")
	p.MarkScheme(8, "inst-8")

	if err := p.flushSchemes(context.Background()); err == nil {
		t.Fatal("expected member 7 load error")
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.scheme"}) {
		t.Fatalf("subjects=%v", got)
	}
	diag := p.Diagnostics()
	if diag.SchemeQueueSize != 1 || diag.Errors == 0 || !strings.Contains(diag.LastError, "member 7") {
		t.Fatalf("diagnostics=%+v", diag)
	}

	source.setSchemeMemberError(7, false)
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.scheme", "caipiao.client.7.scheme"}) {
		t.Fatalf("subjects after retry=%v", got)
	}
}

func TestPublisherPublishFailureRequeuesOnlyFailedMembers(t *testing.T) {
	// Restoring the whole detached batch after one NATS failure republishes
	// already-successful members and couples their progress.
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{
		7: {{ID: "inst-7"}},
		8: {{ID: "inst-8"}},
	}}}
	bus := newRecordingBus()
	bus.failNext("caipiao.client.7.scheme", errors.New("nats unavailable"))
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkScheme(7, "inst-7")
	p.MarkScheme(8, "inst-8")

	if err := p.flushSchemes(context.Background()); err == nil {
		t.Fatal("expected publish error")
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.scheme"}) {
		t.Fatalf("successful subjects=%v", got)
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.scheme", "caipiao.client.7.scheme"}) {
		t.Fatalf("subjects after retry=%v", got)
	}
	if refs := source.lastSchemeRefs(); !reflect.DeepEqual(refs, []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-7"}}) {
		t.Fatalf("retry refs=%v", refs)
	}
}

func TestSuccessfulSchemeFlushMarksStatsDirtyAndPublishesExactStatsWireShape(t *testing.T) {
	// Forgetting the stats invalidation leaves the cloud-center totals stale
	// after a scheme mutation.
	source := &fakeSource{
		schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-7"}}}},
		statsResult: map[int64]schemes.CloudCenterStats{7: {
			GeneratedAt: "2035-04-05T06:07:08.901234567Z",
			Formal:      schemes.CloudCenterChannelStats{TotalTurnover: 12.3, TotalSessionPnl: 4.5},
		}},
	}
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkScheme(7, "inst-7")
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if size := p.Diagnostics().StatsQueueSize; size != 1 {
		t.Fatalf("stats queue size=%d", size)
	}
	if err := p.flushStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.statsCallCount() != 1 || !reflect.DeepEqual(source.lastStatsMemberIDs(), []int64{7}) {
		t.Fatalf("stats calls=%d ids=%v", source.statsCallCount(), source.lastStatsMemberIDs())
	}
	payload, ok := bus.payloadFor("caipiao.client.7.cloud_stats")
	if !ok {
		t.Fatal("stats payload was not published")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	wantFields := []string{"generatedAt", "schemaVersion", "stats"}
	gotFields := make([]string, 0, len(raw))
	for field := range raw {
		gotFields = append(gotFields, field)
	}
	sort.Strings(gotFields)
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Fatalf("wire fields=%v want=%v", gotFields, wantFields)
	}
	var message StatsSnapshotMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatal(err)
	}
	if message.SchemaVersion != 1 ||
		message.GeneratedAt != "2035-04-05T06:07:08.901234567Z" ||
		message.Stats.GeneratedAt != "" ||
		message.Stats.Formal.TotalTurnover != 12.3 {
		t.Fatalf("message=%+v", message)
	}
	if diag := p.Diagnostics(); diag.StatsPublishes != 1 || diag.StatsQueueSize != 0 || diag.LastSuccess.IsZero() || diag.LastPublishLatency < 0 {
		t.Fatalf("diagnostics=%+v", diag)
	}
}

func TestStatsSnapshotWithoutDatabaseVersionIsRejected(t *testing.T) {
	bus := newRecordingBus()
	p := NewPublisher(&fakeSource{}, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkStats(7)
	if got := p.takeStatsDirty(); !reflect.DeepEqual(got, []int64{7}) {
		t.Fatalf("dirty members=%v", got)
	}

	err := p.publishStatsMember(context.Background(), 7, schemes.CloudCenterStats{})
	if err == nil || !strings.Contains(err.Error(), "database generatedAt is required") {
		t.Fatalf("error=%v", err)
	}
	if got := bus.subjects(); len(got) != 0 {
		t.Fatalf("published subjects=%v", got)
	}
	if size := p.Diagnostics().StatsQueueSize; size != 1 {
		t.Fatalf("stats queue size=%d want=1", size)
	}
}

func TestStatsLoadAndPublishFailuresRequeueOnlyAffectedMembers(t *testing.T) {
	// Stats use the same isolation contract as scheme snapshots: a bad member
	// cannot suppress a healthy member, and a publish retry is member-local.
	source := &fakeSource{
		statsResult:     map[int64]schemes.CloudCenterStats{7: {}, 8: {}, 9: {}},
		statsErrMembers: map[int64]bool{7: true},
	}
	bus := newRecordingBus()
	bus.failNext("caipiao.client.8.cloud_stats", errors.New("publish failed"))
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkStats(7)
	p.MarkStats(8)
	p.MarkStats(9)

	if err := p.flushStats(context.Background()); err == nil {
		t.Fatal("expected isolated load and publish errors")
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.9.cloud_stats"}) {
		t.Fatalf("subjects=%v", got)
	}
	if size := p.Diagnostics().StatsQueueSize; size != 2 {
		t.Fatalf("stats queue size=%d", size)
	}

	source.setStatsMemberError(7, false)
	if err := p.flushStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{
		"caipiao.client.9.cloud_stats",
		"caipiao.client.7.cloud_stats",
		"caipiao.client.8.cloud_stats",
	}) {
		t.Fatalf("subjects after retry=%v", got)
	}
}

func TestPublisherSchemeOverflowKeepsNewestDistinctKey(t *testing.T) {
	// An oldest-wins capacity guard leaves inst-a dirty and silently loses the
	// newer committed state for inst-b.
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{
		ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-b"}}},
	}}
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao", MaxSchemeDirty: 1})
	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-b")

	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantRefs := []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-b"}}
	if got := source.lastSchemeRefs(); !reflect.DeepEqual(got, wantRefs) {
		t.Fatalf("loaded refs=%v want=%v", got, wantRefs)
	}
	payload, ok := bus.payloadFor("caipiao.client.7.scheme")
	if !ok {
		t.Fatal("newest scheme key was not published")
	}
	var message SchemeSnapshotMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatal(err)
	}
	if len(message.Items) != 1 || message.Items[0].ID != "inst-b" {
		t.Fatalf("message=%+v", message)
	}
}

func TestPublisherStatsOverflowKeepsNewestDistinctMember(t *testing.T) {
	// A full stats set must retain the newest member, not the first member that
	// happened to become dirty.
	source := &fakeSource{statsResult: map[int64]schemes.CloudCenterStats{8: {}}}
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao", MaxStatsDirty: 1})
	p.MarkStats(7)
	p.MarkStats(8)

	if err := p.flushStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := source.lastStatsMemberIDs(); !reflect.DeepEqual(got, []int64{8}) {
		t.Fatalf("loaded members=%v", got)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.cloud_stats"}) {
		t.Fatalf("subjects=%v", got)
	}
}

func TestPublisherSchemeOverflowDuringFlushKeepsNewestDistinctKey(t *testing.T) {
	// Capacity must remain latest-wins while the only old key is already being
	// loaded, not just while every key is pending.
	source := newBlockingSource()
	source.schemeResult = schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{
		7: {{ID: "inst-a"}, {ID: "inst-b"}},
	}}
	source.blockFirstScheme = true
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao", MaxSchemeDirty: 1})
	p.MarkScheme(7, "inst-a")

	finished := flushSchemesAsync(p, context.Background())
	awaitClosed(t, source.schemeStarted, "scheme source start")
	p.MarkScheme(7, "inst-b")
	close(source.releaseScheme)
	if err := awaitFlush(t, finished); err != nil {
		t.Fatal(err)
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantRefs := []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-b"}}
	if got := source.lastSchemeRefs(); !reflect.DeepEqual(got, wantRefs) {
		t.Fatalf("latest refs=%v want=%v", got, wantRefs)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.7.scheme"}) {
		t.Fatalf("subjects=%v", got)
	}
}

func TestPublisherSameSchemeKeyMarkedDuringSuccessfulFlushRemainsDirty(t *testing.T) {
	// Completing an in-flight key must not clear a newer mark for the same key.
	source := newBlockingSource()
	source.schemeResult = schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-a"}}}}
	source.blockFirstScheme = true
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkScheme(7, "inst-a")

	finished := flushSchemesAsync(p, context.Background())
	awaitClosed(t, source.schemeStarted, "scheme source start")
	p.MarkScheme(7, "inst-a")
	close(source.releaseScheme)
	if err := awaitFlush(t, finished); err != nil {
		t.Fatal(err)
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.schemeCallCount() != 2 || len(bus.subjects()) != 2 {
		t.Fatalf("calls=%d subjects=%v", source.schemeCallCount(), bus.subjects())
	}
}

func TestPublisherSameSchemeKeyMarkedDuringFailedFlushRemainsDirty(t *testing.T) {
	// Restoring a failed in-flight key must merge with, not erase or duplicate,
	// a newer same-key mark.
	source := newBlockingSource()
	source.schemeResult = schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-a"}}}}
	source.blockFirstScheme = true
	source.firstSchemeErr = errors.New("database unavailable")
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkScheme(7, "inst-a")

	finished := flushSchemesAsync(p, context.Background())
	awaitClosed(t, source.schemeStarted, "scheme source start")
	p.MarkScheme(7, "inst-a")
	close(source.releaseScheme)
	if err := awaitFlush(t, finished); err == nil {
		t.Fatal("expected first load failure")
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.schemeCallCount() != 2 || len(bus.subjects()) != 1 {
		t.Fatalf("calls=%d subjects=%v", source.schemeCallCount(), bus.subjects())
	}
}

func TestPublisherSameStatsKeyMarkedDuringSuccessfulFlushRemainsDirty(t *testing.T) {
	source := newBlockingSource()
	source.statsResult = map[int64]schemes.CloudCenterStats{7: {}}
	source.blockFirstStats = true
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkStats(7)

	finished := flushStatsAsync(p, context.Background())
	awaitClosed(t, source.statsStarted, "stats source start")
	p.MarkStats(7)
	close(source.releaseStats)
	if err := awaitFlush(t, finished); err != nil {
		t.Fatal(err)
	}
	if size := p.Diagnostics().StatsQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.statsCallCount() != 2 || len(bus.subjects()) != 2 {
		t.Fatalf("calls=%d subjects=%v", source.statsCallCount(), bus.subjects())
	}
}

func TestPublisherSameStatsKeyMarkedDuringFailedFlushRemainsDirty(t *testing.T) {
	source := newBlockingSource()
	source.statsResult = map[int64]schemes.CloudCenterStats{7: {}}
	source.blockFirstStats = true
	source.firstStatsErr = errors.New("database unavailable")
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	p.MarkStats(7)

	finished := flushStatsAsync(p, context.Background())
	awaitClosed(t, source.statsStarted, "stats source start")
	p.MarkStats(7)
	close(source.releaseStats)
	if err := awaitFlush(t, finished); err == nil {
		t.Fatal("expected first load failure")
	}
	if size := p.Diagnostics().StatsQueueSize; size != 1 {
		t.Fatalf("queue size=%d", size)
	}
	if err := p.flushStats(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.statsCallCount() != 2 || len(bus.subjects()) != 1 {
		t.Fatalf("calls=%d subjects=%v", source.statsCallCount(), bus.subjects())
	}
}

func TestPublisherSchemeCancellationDuringSourceRestoresBatchWithoutFanout(t *testing.T) {
	// A canceled bulk read is terminal for this flush; retrying once per member
	// amplifies cancellation into N additional database calls.
	source := newBlockingSource()
	source.blockFirstScheme = true
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	for memberID := int64(7); memberID <= 9; memberID++ {
		p.MarkScheme(memberID, "inst-"+memberIDString(memberID))
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := flushSchemesAsync(p, ctx)
	awaitClosed(t, source.schemeStarted, "scheme source start")
	cancel()
	err := awaitFlush(t, finished)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if source.schemeCallCount() != 1 {
		t.Fatalf("source calls=%d", source.schemeCallCount())
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 3 {
		t.Fatalf("queue size=%d", size)
	}
	if got := bus.subjects(); len(got) != 0 {
		t.Fatalf("subjects=%v", got)
	}
}

func TestPublisherStatsCancellationDuringSourceRestoresBatchWithoutFanout(t *testing.T) {
	source := newBlockingSource()
	source.blockFirstStats = true
	bus := newRecordingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	for memberID := int64(7); memberID <= 9; memberID++ {
		p.MarkStats(memberID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := flushStatsAsync(p, ctx)
	awaitClosed(t, source.statsStarted, "stats source start")
	cancel()
	err := awaitFlush(t, finished)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if source.statsCallCount() != 1 {
		t.Fatalf("source calls=%d", source.statsCallCount())
	}
	if size := p.Diagnostics().StatsQueueSize; size != 3 {
		t.Fatalf("queue size=%d", size)
	}
	if got := bus.subjects(); len(got) != 0 {
		t.Fatalf("subjects=%v", got)
	}
}

func TestPublisherSchemeCancellationDuringBusRestoresCurrentAndUnprocessedKeys(t *testing.T) {
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{
		7: {{ID: "inst-7"}}, 8: {{ID: "inst-8"}}, 9: {{ID: "inst-9"}},
	}}}
	bus := newBlockingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	for memberID := int64(7); memberID <= 9; memberID++ {
		p.MarkScheme(memberID, "inst-"+memberIDString(memberID))
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := flushSchemesAsync(p, ctx)
	if subject := awaitSubject(t, bus.started); subject != "caipiao.client.7.scheme" {
		t.Fatalf("started subject=%q", subject)
	}
	cancel()
	err := awaitFlush(t, finished)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if got := bus.attemptedSubjects(); !reflect.DeepEqual(got, []string{"caipiao.client.7.scheme"}) {
		t.Fatalf("attempted subjects=%v", got)
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 3 {
		t.Fatalf("queue size=%d", size)
	}
}

func TestPublisherStatsCancellationDuringBusRestoresCurrentAndUnprocessedKeys(t *testing.T) {
	source := &fakeSource{statsResult: map[int64]schemes.CloudCenterStats{7: {}, 8: {}, 9: {}}}
	bus := newBlockingBus()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	for memberID := int64(7); memberID <= 9; memberID++ {
		p.MarkStats(memberID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := flushStatsAsync(p, ctx)
	if subject := awaitSubject(t, bus.started); subject != "caipiao.client.7.cloud_stats" {
		t.Fatalf("started subject=%q", subject)
	}
	cancel()
	err := awaitFlush(t, finished)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if got := bus.attemptedSubjects(); !reflect.DeepEqual(got, []string{"caipiao.client.7.cloud_stats"}) {
		t.Fatalf("attempted subjects=%v", got)
	}
	if size := p.Diagnostics().StatsQueueSize; size != 3 {
		t.Fatalf("queue size=%d", size)
	}
}

func TestPublisherBlockingPublishFailureRestoresOnlyAffectedAndConcurrentlyMarkedKeys(t *testing.T) {
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{
		7: {{ID: "inst-7"}}, 8: {{ID: "inst-8"}}, 9: {{ID: "inst-9"}},
	}}}
	bus := newBlockingBus()
	bus.firstErr = errors.New("nats unavailable")
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao"})
	for memberID := int64(7); memberID <= 9; memberID++ {
		p.MarkScheme(memberID, "inst-"+memberIDString(memberID))
	}
	finished := flushSchemesAsync(p, context.Background())
	if subject := awaitSubject(t, bus.started); subject != "caipiao.client.7.scheme" {
		t.Fatalf("started subject=%q", subject)
	}
	p.MarkScheme(8, "inst-8")
	close(bus.release)
	if err := awaitFlush(t, finished); err == nil {
		t.Fatal("expected member 7 publish failure")
	}
	if size := p.Diagnostics().SchemeQueueSize; size != 2 {
		t.Fatalf("queue size=%d", size)
	}
	if got := bus.subjects(); !reflect.DeepEqual(got, []string{"caipiao.client.8.scheme", "caipiao.client.9.scheme"}) {
		t.Fatalf("successful subjects=%v", got)
	}
	if err := p.flushSchemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantRefs := []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-7"}, {MemberID: 8, InstanceID: "inst-8"}}
	if got := source.lastSchemeRefs(); !reflect.DeepEqual(got, wantRefs) {
		t.Fatalf("retry refs=%v want=%v", got, wantRefs)
	}
}

func TestPublisherBoundsDirtySetsAndReportsOverflow(t *testing.T) {
	// Removing either capacity check permits unbounded growth on a disconnected
	// bus or a database outage.
	p := NewPublisher(&fakeSource{}, newRecordingBus(), Config{
		SubjectPrefix:  "caipiao",
		MaxSchemeDirty: 2,
		MaxStatsDirty:  2,
	})
	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-b")
	p.MarkScheme(7, "inst-c")
	p.MarkScheme(7, "inst-a")
	p.MarkStats(7)
	p.MarkStats(8)
	p.MarkStats(9)
	p.MarkStats(7)

	diag := p.Diagnostics()
	if diag.SchemeQueueSize != 2 || diag.StatsQueueSize != 2 {
		t.Fatalf("diagnostics=%+v", diag)
	}
	if diag.AcceptedSchemeMarks != 4 || diag.CoalescedSchemeMarks != 0 || diag.DroppedSchemeMarks != 2 {
		t.Fatalf("scheme diagnostics=%+v", diag)
	}
	if diag.AcceptedStatsMarks != 4 || diag.CoalescedStatsMarks != 0 || diag.DroppedStatsMarks != 2 {
		t.Fatalf("stats diagnostics=%+v", diag)
	}
}

func TestPublisherDiagnosticsAreRaceSafeDuringMarksAndFlushes(t *testing.T) {
	// Run this test with -race. Separate unsynchronized diagnostic and dirty-set
	// state is otherwise easy to race under worker fan-out.
	source := &fakeSource{
		schemeResult: schemes.RealtimeSchemeSnapshotResult{ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-a"}}}},
		statsResult:  map[int64]schemes.CloudCenterStats{7: {}},
	}
	p := NewPublisher(source, newRecordingBus(), Config{SubjectPrefix: "caipiao", MaxSchemeDirty: 64, MaxStatsDirty: 64})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.MarkScheme(7, "inst-a")
			p.MarkStats(7)
			_ = p.Diagnostics()
		}()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = p.flushSchemes(context.Background())
	}()
	go func() {
		defer wg.Done()
		_ = p.flushStats(context.Background())
	}()
	wg.Wait()
	_ = p.Diagnostics()
}

func TestPublisherRunReturnsWhenContextIsAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := NewPublisher(&fakeSource{}, newRecordingBus(), Config{SubjectPrefix: "caipiao"})
	done := make(chan struct{})
	go func() {
		p.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
}

func TestPublisherSchemeCoalescingTimerCancellationReturnsWithoutFlush(t *testing.T) {
	// The barrier proves cancellation happens after the scheme loop creates its
	// coalescing timer and before that timer fires.
	timers := newBlockingTimerFactory()
	source := &fakeSource{}
	p := NewPublisher(source, newRecordingBus(), Config{SubjectPrefix: "caipiao", newTimer: timers.New})
	p.MarkScheme(7, "inst-a")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.runSchemes(ctx)
		close(done)
	}()
	if duration := timers.awaitStarted(t); duration != 200*time.Millisecond {
		t.Fatalf("duration=%s", duration)
	}
	cancel()
	awaitClosed(t, done, "scheme loop cancellation")
	if source.schemeCallCount() != 0 || p.Diagnostics().SchemeQueueSize != 1 {
		t.Fatalf("calls=%d diagnostics=%+v", source.schemeCallCount(), p.Diagnostics())
	}
}

func TestPublisherStatsCoalescingTimerCancellationReturnsWithoutFlush(t *testing.T) {
	// This independently exercises the one-second stats coalescing timer.
	timers := newBlockingTimerFactory()
	source := &fakeSource{}
	p := NewPublisher(source, newRecordingBus(), Config{SubjectPrefix: "caipiao", newTimer: timers.New})
	p.MarkStats(7)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.runStats(ctx)
		close(done)
	}()
	if duration := timers.awaitStarted(t); duration != time.Second {
		t.Fatalf("duration=%s", duration)
	}
	cancel()
	awaitClosed(t, done, "stats loop cancellation")
	if source.statsCallCount() != 0 || p.Diagnostics().StatsQueueSize != 1 {
		t.Fatalf("calls=%d diagnostics=%+v", source.statsCallCount(), p.Diagnostics())
	}
}

func subscribePayloads(t *testing.T, bus *realtimebus.Memory, subject string) <-chan []byte {
	t.Helper()
	payloads := make(chan []byte, 4)
	if _, err := bus.Subscribe(subject, func(_ string, payload []byte) {
		payloads <- append([]byte(nil), payload...)
	}); err != nil {
		t.Fatal(err)
	}
	return payloads
}

func receiveSchemeMessage(t *testing.T, payloads <-chan []byte) SchemeSnapshotMessage {
	t.Helper()
	select {
	case payload := <-payloads:
		var message SchemeSnapshotMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			t.Fatal(err)
		}
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for scheme message")
		return SchemeSnapshotMessage{}
	}
}

func assertNoPayload(t *testing.T, payloads <-chan []byte) {
	t.Helper()
	select {
	case payload := <-payloads:
		t.Fatalf("unexpected extra payload %s", payload)
	default:
	}
}

func flushSchemesAsync(p *Publisher, ctx context.Context) <-chan error {
	finished := make(chan error, 1)
	go func() { finished <- p.flushSchemes(ctx) }()
	return finished
}

func flushStatsAsync(p *Publisher, ctx context.Context) <-chan error {
	finished := make(chan error, 1)
	go func() { finished <- p.flushStats(ctx) }()
	return finished
}

func awaitClosed(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func awaitSubject(t *testing.T, subjects <-chan string) string {
	t.Helper()
	select {
	case subject := <-subjects:
		return subject
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for publish start")
		return ""
	}
}

func awaitFlush(t *testing.T, finished <-chan error) error {
	t.Helper()
	select {
	case err := <-finished:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for flush")
		return nil
	}
}

type blockingSource struct {
	mu sync.Mutex

	schemeResult schemes.RealtimeSchemeSnapshotResult
	statsResult  map[int64]schemes.CloudCenterStats

	blockFirstScheme bool
	blockFirstStats  bool
	firstSchemeErr   error
	firstStatsErr    error
	schemeStarted    chan struct{}
	statsStarted     chan struct{}
	releaseScheme    chan struct{}
	releaseStats     chan struct{}
	schemeCalls      int
	statsCalls       int
	lastRefs         []schemes.RealtimeInstanceRef
	lastMembers      []int64
}

const fakeStatsGeneratedAt = "2000-01-01T00:00:00Z"

func withFakeStatsGeneratedAt(result map[int64]schemes.CloudCenterStats) map[int64]schemes.CloudCenterStats {
	cloned := make(map[int64]schemes.CloudCenterStats, len(result))
	for memberID, stats := range result {
		if stats.GeneratedAt == "" {
			stats.GeneratedAt = fakeStatsGeneratedAt
		}
		cloned[memberID] = stats
	}
	return cloned
}

type blockingTimerFactory struct {
	started chan time.Duration
	fire    chan time.Time
}

func newBlockingTimerFactory() *blockingTimerFactory {
	return &blockingTimerFactory{started: make(chan time.Duration, 1), fire: make(chan time.Time)}
}

func (f *blockingTimerFactory) New(duration time.Duration) (<-chan time.Time, func()) {
	f.started <- duration
	return f.fire, func() {}
}

func (f *blockingTimerFactory) awaitStarted(t *testing.T) time.Duration {
	t.Helper()
	select {
	case duration := <-f.started:
		return duration
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for coalescing timer")
		return 0
	}
}

func newBlockingSource() *blockingSource {
	return &blockingSource{
		schemeStarted: make(chan struct{}),
		statsStarted:  make(chan struct{}),
		releaseScheme: make(chan struct{}),
		releaseStats:  make(chan struct{}),
	}
}

func (s *blockingSource) LoadRealtimeSchemeSnapshots(ctx context.Context, refs []schemes.RealtimeInstanceRef) (schemes.RealtimeSchemeSnapshotResult, error) {
	s.mu.Lock()
	s.schemeCalls++
	call := s.schemeCalls
	s.lastRefs = append([]schemes.RealtimeInstanceRef(nil), refs...)
	block := s.blockFirstScheme && call == 1
	result := s.schemeResult
	err := error(nil)
	if call == 1 {
		err = s.firstSchemeErr
	}
	s.mu.Unlock()
	if block {
		close(s.schemeStarted)
		select {
		case <-s.releaseScheme:
		case <-ctx.Done():
			return schemes.RealtimeSchemeSnapshotResult{}, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return schemes.RealtimeSchemeSnapshotResult{}, err
	}
	return result, err
}

func (s *blockingSource) LoadRealtimeStats(ctx context.Context, memberIDs []int64) (map[int64]schemes.CloudCenterStats, error) {
	s.mu.Lock()
	s.statsCalls++
	call := s.statsCalls
	s.lastMembers = append([]int64(nil), memberIDs...)
	block := s.blockFirstStats && call == 1
	result := s.statsResult
	err := error(nil)
	if call == 1 {
		err = s.firstStatsErr
	}
	s.mu.Unlock()
	if block {
		close(s.statsStarted)
		select {
		case <-s.releaseStats:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return withFakeStatsGeneratedAt(result), err
}

func (s *blockingSource) schemeCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.schemeCalls
}

func (s *blockingSource) statsCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statsCalls
}

func (s *blockingSource) lastSchemeRefs() []schemes.RealtimeInstanceRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]schemes.RealtimeInstanceRef(nil), s.lastRefs...)
}

type fakeSource struct {
	mu sync.Mutex

	schemeResult     schemes.RealtimeSchemeSnapshotResult
	statsResult      map[int64]schemes.CloudCenterStats
	schemeErrMembers map[int64]bool
	statsErrMembers  map[int64]bool

	schemeCalls int
	statsCalls  int
	lastRefs    []schemes.RealtimeInstanceRef
	lastMembers []int64
}

func (s *fakeSource) LoadRealtimeSchemeSnapshots(_ context.Context, refs []schemes.RealtimeInstanceRef) (schemes.RealtimeSchemeSnapshotResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schemeCalls++
	s.lastRefs = append([]schemes.RealtimeInstanceRef(nil), refs...)
	for _, ref := range refs {
		if s.schemeErrMembers[ref.MemberID] {
			return schemes.RealtimeSchemeSnapshotResult{}, errors.New("load scheme member " + memberIDString(ref.MemberID))
		}
	}
	return s.schemeResult, nil
}

func (s *fakeSource) LoadRealtimeStats(_ context.Context, memberIDs []int64) (map[int64]schemes.CloudCenterStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsCalls++
	s.lastMembers = append([]int64(nil), memberIDs...)
	for _, memberID := range memberIDs {
		if s.statsErrMembers[memberID] {
			return nil, errors.New("load stats member " + memberIDString(memberID))
		}
	}
	return withFakeStatsGeneratedAt(s.statsResult), nil
}

func (s *fakeSource) schemeCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.schemeCalls
}

func (s *fakeSource) statsCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statsCalls
}

func (s *fakeSource) lastSchemeRefs() []schemes.RealtimeInstanceRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]schemes.RealtimeInstanceRef(nil), s.lastRefs...)
}

func (s *fakeSource) lastStatsMemberIDs() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.lastMembers...)
}

func (s *fakeSource) setSchemeMemberError(memberID int64, fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schemeErrMembers == nil {
		s.schemeErrMembers = make(map[int64]bool)
	}
	s.schemeErrMembers[memberID] = fail
}

func (s *fakeSource) setStatsMemberError(memberID int64, fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.statsErrMembers == nil {
		s.statsErrMembers = make(map[int64]bool)
	}
	s.statsErrMembers[memberID] = fail
}

type recordedMessage struct {
	subject string
	payload []byte
}

type recordingBus struct {
	mu       sync.Mutex
	messages []recordedMessage
	failures map[string][]error
}

type blockingBus struct {
	mu       sync.Mutex
	attempts []string
	messages []recordedMessage
	started  chan string
	release  chan struct{}
	firstErr error
}

func newBlockingBus() *blockingBus {
	return &blockingBus{started: make(chan string, 1), release: make(chan struct{})}
}

func (b *blockingBus) Publish(ctx context.Context, subject string, payload []byte) error {
	b.mu.Lock()
	b.attempts = append(b.attempts, subject)
	call := len(b.attempts)
	b.mu.Unlock()
	if call == 1 {
		b.started <- subject
		select {
		case <-b.release:
		case <-ctx.Done():
			return ctx.Err()
		}
		if b.firstErr != nil {
			return b.firstErr
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	b.messages = append(b.messages, recordedMessage{subject: subject, payload: append([]byte(nil), payload...)})
	b.mu.Unlock()
	return nil
}

func (b *blockingBus) Subscribe(string, realtimebus.Handler) (realtimebus.Subscription, error) {
	return nil, errors.New("blocking bus does not subscribe")
}

func (b *blockingBus) OnConnectionChange(func(bool)) {}
func (b *blockingBus) Diagnostics() realtimebus.Diagnostics {
	return realtimebus.Diagnostics{Kind: "blocking", Connected: true}
}
func (b *blockingBus) Close() error { return nil }

func (b *blockingBus) attemptedSubjects() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.attempts...)
}

func (b *blockingBus) subjects() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, 0, len(b.messages))
	for _, message := range b.messages {
		result = append(result, message.subject)
	}
	return result
}

func newRecordingBus() *recordingBus {
	return &recordingBus{failures: make(map[string][]error)}
}

func (b *recordingBus) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if failures := b.failures[subject]; len(failures) > 0 {
		err := failures[0]
		b.failures[subject] = failures[1:]
		return err
	}
	b.messages = append(b.messages, recordedMessage{subject: subject, payload: append([]byte(nil), payload...)})
	return nil
}

func (b *recordingBus) Subscribe(string, realtimebus.Handler) (realtimebus.Subscription, error) {
	return nil, errors.New("recording bus does not subscribe")
}

func (b *recordingBus) OnConnectionChange(func(bool)) {}

func (b *recordingBus) Diagnostics() realtimebus.Diagnostics {
	return realtimebus.Diagnostics{Kind: "recording", Connected: true}
}

func (b *recordingBus) Close() error { return nil }

func (b *recordingBus) failNext(subject string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures[subject] = append(b.failures[subject], err)
}

func (b *recordingBus) subjects() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, 0, len(b.messages))
	for _, message := range b.messages {
		result = append(result, message.subject)
	}
	return result
}

func (b *recordingBus) payloadFor(subject string) ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := len(b.messages) - 1; i >= 0; i-- {
		if b.messages[i].subject == subject {
			return append([]byte(nil), b.messages[i].payload...), true
		}
	}
	return nil, false
}

func memberIDString(memberID int64) string {
	const digits = "0123456789"
	if memberID == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for memberID > 0 {
		i--
		buf[i] = digits[memberID%10]
		memberID /= 10
	}
	return string(buf[i:])
}
