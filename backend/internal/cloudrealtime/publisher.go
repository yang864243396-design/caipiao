package cloudrealtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/schemeevents"
	"caipiao/backend/internal/schemes"
)

const (
	defaultSchemeCoalesce = 200 * time.Millisecond
	defaultStatsCoalesce  = time.Second
	defaultMaxSchemeDirty = 4096
	defaultMaxStatsDirty  = 4096
)

type Config struct {
	SubjectPrefix  string
	SchemeCoalesce time.Duration
	StatsCoalesce  time.Duration
	MaxSchemeDirty int
	MaxStatsDirty  int
	newTimer       func(time.Duration) (<-chan time.Time, func())
}

type schemeKey struct {
	memberID   int64
	instanceID string
}

type Publisher struct {
	source SnapshotSource
	bus    realtimebus.Bus
	cfg    Config

	mu             sync.Mutex
	schemeDirty    map[schemeKey]uint64
	schemeInFlight map[schemeKey]uint64
	statsDirty     map[int64]uint64
	statsInFlight  map[int64]uint64
	nextRecency    uint64
	diagnostics    Diagnostics
	schemeWake     chan struct{}
	statsWake      chan struct{}
	schemeFlushMu  sync.Mutex
	statsFlushMu   sync.Mutex
}

var _ schemeevents.Marker = (*Publisher)(nil)

func NewPublisher(source SnapshotSource, bus realtimebus.Bus, cfg Config) *Publisher {
	if cfg.SchemeCoalesce <= 0 {
		cfg.SchemeCoalesce = defaultSchemeCoalesce
	}
	if cfg.StatsCoalesce <= 0 {
		cfg.StatsCoalesce = defaultStatsCoalesce
	}
	if cfg.MaxSchemeDirty <= 0 {
		cfg.MaxSchemeDirty = defaultMaxSchemeDirty
	}
	if cfg.MaxStatsDirty <= 0 {
		cfg.MaxStatsDirty = defaultMaxStatsDirty
	}
	if cfg.newTimer == nil {
		cfg.newTimer = newCoalescingTimer
	}
	return &Publisher{
		source:         source,
		bus:            bus,
		cfg:            cfg,
		schemeDirty:    make(map[schemeKey]uint64),
		schemeInFlight: make(map[schemeKey]uint64),
		statsDirty:     make(map[int64]uint64),
		statsInFlight:  make(map[int64]uint64),
		schemeWake:     make(chan struct{}, 1),
		statsWake:      make(chan struct{}, 1),
	}
}

func (p *Publisher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p.runSchemes(ctx)
	}()
	go func() {
		defer wg.Done()
		p.runStats(ctx)
	}()
	wg.Wait()
}

func (p *Publisher) MarkScheme(memberID int64, instanceID string) {
	instanceID = strings.TrimSpace(instanceID)
	if memberID <= 0 || instanceID == "" {
		return
	}
	key := schemeKey{memberID: memberID, instanceID: instanceID}

	p.mu.Lock()
	_, pending := p.schemeDirty[key]
	_, inFlight := p.schemeInFlight[key]
	if pending {
		p.schemeDirty[key] = p.nextRecencyLocked()
		p.diagnostics.AcceptedSchemeMarks++
		p.diagnostics.CoalescedSchemeMarks++
		p.mu.Unlock()
		p.wakeSchemes()
		return
	}
	if inFlight {
		p.schemeDirty[key] = p.nextRecencyLocked()
		p.diagnostics.AcceptedSchemeMarks++
		p.diagnostics.CoalescedSchemeMarks++
		p.mu.Unlock()
		p.wakeSchemes()
		return
	}
	if p.schemeOutstandingLocked() >= p.cfg.MaxSchemeDirty {
		p.evictOldestSchemeLocked()
		p.diagnostics.DroppedSchemeMarks++
	}
	p.schemeDirty[key] = p.nextRecencyLocked()
	p.diagnostics.AcceptedSchemeMarks++
	p.mu.Unlock()
	p.wakeSchemes()
}

func (p *Publisher) MarkStats(memberID int64) {
	if memberID <= 0 {
		return
	}
	p.mu.Lock()
	_, pending := p.statsDirty[memberID]
	_, inFlight := p.statsInFlight[memberID]
	if pending {
		p.statsDirty[memberID] = p.nextRecencyLocked()
		p.diagnostics.AcceptedStatsMarks++
		p.diagnostics.CoalescedStatsMarks++
		p.mu.Unlock()
		p.wakeStats()
		return
	}
	if inFlight {
		p.statsDirty[memberID] = p.nextRecencyLocked()
		p.diagnostics.AcceptedStatsMarks++
		p.diagnostics.CoalescedStatsMarks++
		p.mu.Unlock()
		p.wakeStats()
		return
	}
	if p.statsOutstandingLocked() >= p.cfg.MaxStatsDirty {
		p.evictOldestStatsLocked()
		p.diagnostics.DroppedStatsMarks++
	}
	p.statsDirty[memberID] = p.nextRecencyLocked()
	p.diagnostics.AcceptedStatsMarks++
	p.mu.Unlock()
	p.wakeStats()
}

func (p *Publisher) Diagnostics() Diagnostics {
	p.mu.Lock()
	defer p.mu.Unlock()
	diagnostics := p.diagnostics
	diagnostics.SchemeQueueSize = p.schemeOutstandingLocked()
	diagnostics.StatsQueueSize = p.statsOutstandingLocked()
	return diagnostics
}

func (p *Publisher) runSchemes(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.schemeWake:
		}
		if !waitForWindow(ctx, p.cfg.SchemeCoalesce, p.cfg.newTimer) {
			return
		}
		_ = p.flushSchemes(ctx)
	}
}

func (p *Publisher) runStats(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.statsWake:
		}
		if !waitForWindow(ctx, p.cfg.StatsCoalesce, p.cfg.newTimer) {
			return
		}
		_ = p.flushStats(ctx)
	}
}

func waitForWindow(ctx context.Context, duration time.Duration, newTimer func(time.Duration) (<-chan time.Time, func())) bool {
	ticks, stop := newTimer(duration)
	defer stop()
	select {
	case <-ctx.Done():
		return false
	case <-ticks:
		return true
	}
}

func newCoalescingTimer(duration time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(duration)
	return timer.C, func() { timer.Stop() }
}

func (p *Publisher) flushSchemes(ctx context.Context) error {
	p.schemeFlushMu.Lock()
	defer p.schemeFlushMu.Unlock()

	keys := p.takeSchemeDirty()
	if len(keys) == 0 {
		return nil
	}
	groups := groupSchemeKeys(keys)
	refs := refsForSchemeKeys(keys)
	if p.source == nil {
		err := errors.New("scheme snapshot source is required")
		p.restoreSchemeKeys(keys)
		p.recordError(err)
		return err
	}
	if err := ctx.Err(); err != nil {
		p.restoreSchemeKeys(keys)
		p.recordError(err)
		return err
	}

	result, loadErr := p.source.LoadRealtimeSchemeSnapshots(ctx, refs)
	if loadErr == nil {
		return p.publishSchemeGroups(ctx, groups, result)
	}
	if canceled := cancellationError(ctx, loadErr); canceled != nil {
		err := fmt.Errorf("load scheme snapshots: %w", canceled)
		p.restoreSchemeKeys(keys)
		p.recordError(err)
		return err
	}

	initialErr := fmt.Errorf("load scheme snapshots: %w", loadErr)
	p.recordError(initialErr)
	if len(groups) == 1 {
		p.restoreSchemeKeys(keys)
		return initialErr
	}

	errs := []error{initialErr}
	memberIDs := sortedSchemeMemberIDs(groups)
	for index, memberID := range memberIDs {
		if canceled := ctx.Err(); canceled != nil {
			p.restoreSchemeKeys(schemeKeysForMembers(groups, memberIDs[index:]))
			p.recordError(canceled)
			return errors.Join(append(errs, canceled)...)
		}
		memberKeys := p.activeSchemeKeys(groups[memberID])
		if len(memberKeys) == 0 {
			continue
		}
		memberResult, err := p.source.LoadRealtimeSchemeSnapshots(ctx, refsForSchemeKeys(memberKeys))
		if err != nil {
			if canceled := cancellationError(ctx, err); canceled != nil {
				memberErr := fmt.Errorf("load scheme snapshots for member %d: %w", memberID, canceled)
				p.restoreSchemeKeys(schemeKeysForMembers(groups, memberIDs[index:]))
				p.recordError(memberErr)
				return errors.Join(append(errs, memberErr)...)
			}
			memberErr := fmt.Errorf("load scheme snapshots for member %d: %w", memberID, err)
			p.restoreSchemeKeys(memberKeys)
			p.recordError(memberErr)
			errs = append(errs, memberErr)
			continue
		}
		if err := p.publishSchemeMember(ctx, memberID, memberKeys, memberResult); err != nil {
			if cancellationError(ctx, err) != nil {
				p.restoreSchemeKeys(schemeKeysForMembers(groups, memberIDs[index:]))
				return errors.Join(append(errs, err)...)
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishSchemeGroups(ctx context.Context, groups map[int64][]schemeKey, result schemes.RealtimeSchemeSnapshotResult) error {
	var errs []error
	memberIDs := sortedSchemeMemberIDs(groups)
	for index, memberID := range memberIDs {
		if canceled := ctx.Err(); canceled != nil {
			p.restoreSchemeKeys(schemeKeysForMembers(groups, memberIDs[index:]))
			p.recordError(canceled)
			return errors.Join(append(errs, canceled)...)
		}
		if err := p.publishSchemeMember(ctx, memberID, groups[memberID], result); err != nil {
			if cancellationError(ctx, err) != nil {
				p.restoreSchemeKeys(schemeKeysForMembers(groups, memberIDs[index:]))
				return errors.Join(append(errs, err)...)
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishSchemeMember(ctx context.Context, memberID int64, keys []schemeKey, result schemes.RealtimeSchemeSnapshotResult) error {
	keys = p.activeSchemeKeys(keys)
	if len(keys) == 0 {
		return nil
	}
	subject, err := SchemeSubject(p.cfg.SubjectPrefix, memberID)
	if err != nil {
		return p.failSchemeMember(keys, fmt.Errorf("scheme subject for member %d: %w", memberID, err))
	}
	message := SchemeSnapshotMessage{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Items:         schemeItemsForKeys(result.ItemsByMember[memberID], keys),
		RemovedIDs:    removedIDsForKeys(result.RemovedByMember[memberID], keys),
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return p.failSchemeMember(keys, fmt.Errorf("marshal scheme snapshot for member %d: %w", memberID, err))
	}
	if p.bus == nil {
		return p.failSchemeMember(keys, fmt.Errorf("publish scheme snapshot for member %d: realtime bus is required", memberID))
	}
	started := time.Now()
	err = p.bus.Publish(ctx, subject, payload)
	latency := time.Since(started)
	if err != nil {
		p.recordPublishLatency(latency)
		publishErr := fmt.Errorf("publish scheme snapshot for member %d: %w", memberID, err)
		if cancellationError(ctx, err) != nil {
			p.recordError(publishErr)
			return publishErr
		}
		return p.failSchemeMember(keys, publishErr)
	}
	p.completeSchemeKeys(keys)
	p.recordPublishSuccess(true, latency)
	p.MarkStats(memberID)
	return nil
}

func (p *Publisher) failSchemeMember(keys []schemeKey, err error) error {
	p.restoreSchemeKeys(keys)
	p.recordError(err)
	return err
}

func (p *Publisher) flushStats(ctx context.Context) error {
	p.statsFlushMu.Lock()
	defer p.statsFlushMu.Unlock()

	memberIDs := p.takeStatsDirty()
	if len(memberIDs) == 0 {
		return nil
	}
	if p.source == nil {
		err := errors.New("stats snapshot source is required")
		p.restoreStatsMembers(memberIDs)
		p.recordError(err)
		return err
	}
	if err := ctx.Err(); err != nil {
		p.restoreStatsMembers(memberIDs)
		p.recordError(err)
		return err
	}

	result, loadErr := p.source.LoadRealtimeStats(ctx, memberIDs)
	if loadErr == nil {
		return p.publishStatsMembers(ctx, memberIDs, result)
	}
	if canceled := cancellationError(ctx, loadErr); canceled != nil {
		err := fmt.Errorf("load stats snapshots: %w", canceled)
		p.restoreStatsMembers(memberIDs)
		p.recordError(err)
		return err
	}

	initialErr := fmt.Errorf("load stats snapshots: %w", loadErr)
	p.recordError(initialErr)
	if len(memberIDs) == 1 {
		p.restoreStatsMembers(memberIDs)
		return initialErr
	}

	errs := []error{initialErr}
	for index, memberID := range memberIDs {
		if canceled := ctx.Err(); canceled != nil {
			p.restoreStatsMembers(memberIDs[index:])
			p.recordError(canceled)
			return errors.Join(append(errs, canceled)...)
		}
		if !p.statsMemberInFlight(memberID) {
			continue
		}
		memberResult, err := p.source.LoadRealtimeStats(ctx, []int64{memberID})
		if err != nil {
			if canceled := cancellationError(ctx, err); canceled != nil {
				memberErr := fmt.Errorf("load stats snapshot for member %d: %w", memberID, canceled)
				p.restoreStatsMembers(memberIDs[index:])
				p.recordError(memberErr)
				return errors.Join(append(errs, memberErr)...)
			}
			memberErr := fmt.Errorf("load stats snapshot for member %d: %w", memberID, err)
			p.restoreStatsMembers([]int64{memberID})
			p.recordError(memberErr)
			errs = append(errs, memberErr)
			continue
		}
		if err := p.publishStatsMember(ctx, memberID, memberResult[memberID]); err != nil {
			if cancellationError(ctx, err) != nil {
				p.restoreStatsMembers(memberIDs[index:])
				return errors.Join(append(errs, err)...)
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishStatsMembers(ctx context.Context, memberIDs []int64, result map[int64]schemes.CloudCenterStats) error {
	var errs []error
	for index, memberID := range memberIDs {
		if canceled := ctx.Err(); canceled != nil {
			p.restoreStatsMembers(memberIDs[index:])
			p.recordError(canceled)
			return errors.Join(append(errs, canceled)...)
		}
		if err := p.publishStatsMember(ctx, memberID, result[memberID]); err != nil {
			if cancellationError(ctx, err) != nil {
				p.restoreStatsMembers(memberIDs[index:])
				return errors.Join(append(errs, err)...)
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishStatsMember(ctx context.Context, memberID int64, stats schemes.CloudCenterStats) error {
	if !p.statsMemberInFlight(memberID) {
		return nil
	}
	subject, err := StatsSubject(p.cfg.SubjectPrefix, memberID)
	if err != nil {
		return p.failStatsMember(memberID, fmt.Errorf("stats subject for member %d: %w", memberID, err))
	}
	generatedAt := strings.TrimSpace(stats.GeneratedAt)
	if generatedAt == "" {
		return p.failStatsMember(memberID, fmt.Errorf("stats snapshot for member %d: database generatedAt is required", memberID))
	}
	stats.GeneratedAt = ""
	message := StatsSnapshotMessage{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt,
		Stats:         stats,
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return p.failStatsMember(memberID, fmt.Errorf("marshal stats snapshot for member %d: %w", memberID, err))
	}
	if p.bus == nil {
		return p.failStatsMember(memberID, fmt.Errorf("publish stats snapshot for member %d: realtime bus is required", memberID))
	}
	started := time.Now()
	err = p.bus.Publish(ctx, subject, payload)
	latency := time.Since(started)
	if err != nil {
		p.recordPublishLatency(latency)
		publishErr := fmt.Errorf("publish stats snapshot for member %d: %w", memberID, err)
		if cancellationError(ctx, err) != nil {
			p.recordError(publishErr)
			return publishErr
		}
		return p.failStatsMember(memberID, publishErr)
	}
	p.completeStatsMember(memberID)
	p.recordPublishSuccess(false, latency)
	return nil
}

func (p *Publisher) failStatsMember(memberID int64, err error) error {
	p.restoreStatsMembers([]int64{memberID})
	p.recordError(err)
	return err
}

func (p *Publisher) takeSchemeDirty() []schemeKey {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make([]schemeKey, 0, len(p.schemeDirty))
	for key, recency := range p.schemeDirty {
		keys = append(keys, key)
		p.schemeInFlight[key] = recency
	}
	p.schemeDirty = make(map[schemeKey]uint64)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].memberID != keys[j].memberID {
			return keys[i].memberID < keys[j].memberID
		}
		return keys[i].instanceID < keys[j].instanceID
	})
	return keys
}

func (p *Publisher) takeStatsDirty() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	memberIDs := make([]int64, 0, len(p.statsDirty))
	for memberID, recency := range p.statsDirty {
		memberIDs = append(memberIDs, memberID)
		p.statsInFlight[memberID] = recency
	}
	p.statsDirty = make(map[int64]uint64)
	sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
	return memberIDs
}

func (p *Publisher) restoreSchemeKeys(keys []schemeKey) {
	p.mu.Lock()
	for _, key := range keys {
		recency, active := p.schemeInFlight[key]
		if !active {
			continue
		}
		delete(p.schemeInFlight, key)
		if _, newer := p.schemeDirty[key]; !newer {
			p.schemeDirty[key] = recency
		}
	}
	p.mu.Unlock()
	p.wakeSchemes()
}

func (p *Publisher) completeSchemeKeys(keys []schemeKey) {
	p.mu.Lock()
	for _, key := range keys {
		delete(p.schemeInFlight, key)
	}
	p.mu.Unlock()
}

func (p *Publisher) restoreStatsMembers(memberIDs []int64) {
	p.mu.Lock()
	for _, memberID := range memberIDs {
		recency, active := p.statsInFlight[memberID]
		if !active {
			continue
		}
		delete(p.statsInFlight, memberID)
		if _, newer := p.statsDirty[memberID]; !newer {
			p.statsDirty[memberID] = recency
		}
	}
	p.mu.Unlock()
	p.wakeStats()
}

func (p *Publisher) completeStatsMember(memberID int64) {
	p.mu.Lock()
	delete(p.statsInFlight, memberID)
	p.mu.Unlock()
}

func (p *Publisher) recordPublishSuccess(scheme bool, latency time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if scheme {
		p.diagnostics.SchemePublishes++
	} else {
		p.diagnostics.StatsPublishes++
	}
	p.diagnostics.LastSuccess = time.Now().UTC()
	p.diagnostics.LastPublishLatency = latency
}

func (p *Publisher) recordPublishLatency(latency time.Duration) {
	p.mu.Lock()
	p.diagnostics.LastPublishLatency = latency
	p.mu.Unlock()
}

func (p *Publisher) recordError(err error) {
	p.mu.Lock()
	p.diagnostics.Errors++
	p.diagnostics.LastError = err.Error()
	p.mu.Unlock()
}

func (p *Publisher) schemeOutstandingLocked() int {
	count := len(p.schemeInFlight)
	for key := range p.schemeDirty {
		if _, exists := p.schemeInFlight[key]; !exists {
			count++
		}
	}
	return count
}

func (p *Publisher) statsOutstandingLocked() int {
	count := len(p.statsInFlight)
	for memberID := range p.statsDirty {
		if _, exists := p.statsInFlight[memberID]; !exists {
			count++
		}
	}
	return count
}

func (p *Publisher) nextRecencyLocked() uint64 {
	p.nextRecency++
	return p.nextRecency
}

func (p *Publisher) evictOldestSchemeLocked() {
	var oldest schemeKey
	var oldestRecency uint64
	found := false
	for key, recency := range p.schemeInFlight {
		if pendingRecency, pending := p.schemeDirty[key]; pending {
			recency = pendingRecency
		}
		if !found || recency < oldestRecency {
			oldest, oldestRecency, found = key, recency, true
		}
	}
	for key, recency := range p.schemeDirty {
		if _, inFlight := p.schemeInFlight[key]; inFlight {
			continue
		}
		if !found || recency < oldestRecency {
			oldest, oldestRecency, found = key, recency, true
		}
	}
	if found {
		delete(p.schemeDirty, oldest)
		delete(p.schemeInFlight, oldest)
	}
}

func (p *Publisher) evictOldestStatsLocked() {
	var oldest int64
	var oldestRecency uint64
	found := false
	for memberID, recency := range p.statsInFlight {
		if pendingRecency, pending := p.statsDirty[memberID]; pending {
			recency = pendingRecency
		}
		if !found || recency < oldestRecency {
			oldest, oldestRecency, found = memberID, recency, true
		}
	}
	for memberID, recency := range p.statsDirty {
		if _, inFlight := p.statsInFlight[memberID]; inFlight {
			continue
		}
		if !found || recency < oldestRecency {
			oldest, oldestRecency, found = memberID, recency, true
		}
	}
	if found {
		delete(p.statsDirty, oldest)
		delete(p.statsInFlight, oldest)
	}
}

func (p *Publisher) activeSchemeKeys(keys []schemeKey) []schemeKey {
	p.mu.Lock()
	defer p.mu.Unlock()
	active := make([]schemeKey, 0, len(keys))
	for _, key := range keys {
		if _, ok := p.schemeInFlight[key]; ok {
			active = append(active, key)
		}
	}
	return active
}

func (p *Publisher) statsMemberInFlight(memberID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, active := p.statsInFlight[memberID]
	return active
}

func (p *Publisher) wakeSchemes() {
	select {
	case p.schemeWake <- struct{}{}:
	default:
	}
}

func (p *Publisher) wakeStats() {
	select {
	case p.statsWake <- struct{}{}:
	default:
	}
}

func groupSchemeKeys(keys []schemeKey) map[int64][]schemeKey {
	groups := make(map[int64][]schemeKey)
	for _, key := range keys {
		groups[key.memberID] = append(groups[key.memberID], key)
	}
	return groups
}

func refsForSchemeKeys(keys []schemeKey) []schemes.RealtimeInstanceRef {
	refs := make([]schemes.RealtimeInstanceRef, 0, len(keys))
	for _, key := range keys {
		refs = append(refs, schemes.RealtimeInstanceRef{MemberID: key.memberID, InstanceID: key.instanceID})
	}
	return refs
}

func sortedSchemeMemberIDs(groups map[int64][]schemeKey) []int64 {
	memberIDs := make([]int64, 0, len(groups))
	for memberID := range groups {
		memberIDs = append(memberIDs, memberID)
	}
	sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
	return memberIDs
}

func schemeKeysForMembers(groups map[int64][]schemeKey, memberIDs []int64) []schemeKey {
	var keys []schemeKey
	for _, memberID := range memberIDs {
		keys = append(keys, groups[memberID]...)
	}
	return keys
}

func cancellationError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

func schemeItemsForKeys(items []schemes.Instance, keys []schemeKey) []schemes.Instance {
	instanceIDs := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		instanceIDs[key.instanceID] = struct{}{}
	}
	filtered := make([]schemes.Instance, 0, len(items))
	for _, item := range items {
		if _, active := instanceIDs[item.ID]; active {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func removedIDsForKeys(items []string, keys []schemeKey) []string {
	instanceIDs := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		instanceIDs[key.instanceID] = struct{}{}
	}
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if _, active := instanceIDs[item]; active {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
