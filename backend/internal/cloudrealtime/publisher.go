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
	schemeDirty    map[schemeKey]struct{}
	schemeInFlight map[schemeKey]struct{}
	statsDirty     map[int64]struct{}
	statsInFlight  map[int64]struct{}
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
	return &Publisher{
		source:         source,
		bus:            bus,
		cfg:            cfg,
		schemeDirty:    make(map[schemeKey]struct{}),
		schemeInFlight: make(map[schemeKey]struct{}),
		statsDirty:     make(map[int64]struct{}),
		statsInFlight:  make(map[int64]struct{}),
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
		p.diagnostics.AcceptedSchemeMarks++
		p.diagnostics.CoalescedSchemeMarks++
		p.mu.Unlock()
		p.wakeSchemes()
		return
	}
	if !inFlight && p.schemeOutstandingLocked() >= p.cfg.MaxSchemeDirty {
		p.diagnostics.DroppedSchemeMarks++
		p.mu.Unlock()
		return
	}
	p.schemeDirty[key] = struct{}{}
	p.diagnostics.AcceptedSchemeMarks++
	if inFlight {
		p.diagnostics.CoalescedSchemeMarks++
	}
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
		p.diagnostics.AcceptedStatsMarks++
		p.diagnostics.CoalescedStatsMarks++
		p.mu.Unlock()
		p.wakeStats()
		return
	}
	if !inFlight && p.statsOutstandingLocked() >= p.cfg.MaxStatsDirty {
		p.diagnostics.DroppedStatsMarks++
		p.mu.Unlock()
		return
	}
	p.statsDirty[memberID] = struct{}{}
	p.diagnostics.AcceptedStatsMarks++
	if inFlight {
		p.diagnostics.CoalescedStatsMarks++
	}
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
		if !waitForWindow(ctx, p.cfg.SchemeCoalesce) {
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
		if !waitForWindow(ctx, p.cfg.StatsCoalesce) {
			return
		}
		_ = p.flushStats(ctx)
	}
}

func waitForWindow(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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

	result, loadErr := p.source.LoadRealtimeSchemeSnapshots(ctx, refs)
	if loadErr == nil {
		return p.publishSchemeGroups(ctx, groups, result)
	}

	initialErr := fmt.Errorf("load scheme snapshots: %w", loadErr)
	p.recordError(initialErr)
	if len(groups) == 1 {
		p.restoreSchemeKeys(keys)
		return initialErr
	}

	errs := []error{initialErr}
	memberIDs := sortedSchemeMemberIDs(groups)
	for _, memberID := range memberIDs {
		memberKeys := groups[memberID]
		memberResult, err := p.source.LoadRealtimeSchemeSnapshots(ctx, refsForSchemeKeys(memberKeys))
		if err != nil {
			memberErr := fmt.Errorf("load scheme snapshots for member %d: %w", memberID, err)
			p.restoreSchemeKeys(memberKeys)
			p.recordError(memberErr)
			errs = append(errs, memberErr)
			continue
		}
		if err := p.publishSchemeMember(ctx, memberID, memberKeys, memberResult); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishSchemeGroups(ctx context.Context, groups map[int64][]schemeKey, result schemes.RealtimeSchemeSnapshotResult) error {
	var errs []error
	for _, memberID := range sortedSchemeMemberIDs(groups) {
		if err := p.publishSchemeMember(ctx, memberID, groups[memberID], result); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishSchemeMember(ctx context.Context, memberID int64, keys []schemeKey, result schemes.RealtimeSchemeSnapshotResult) error {
	subject, err := SchemeSubject(p.cfg.SubjectPrefix, memberID)
	if err != nil {
		return p.failSchemeMember(keys, fmt.Errorf("scheme subject for member %d: %w", memberID, err))
	}
	message := SchemeSnapshotMessage{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Items:         nonNilInstances(result.ItemsByMember[memberID]),
		RemovedIDs:    nonNilStrings(result.RemovedByMember[memberID]),
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
		return p.failSchemeMember(keys, fmt.Errorf("publish scheme snapshot for member %d: %w", memberID, err))
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

	result, loadErr := p.source.LoadRealtimeStats(ctx, memberIDs)
	if loadErr == nil {
		return p.publishStatsMembers(ctx, memberIDs, result)
	}

	initialErr := fmt.Errorf("load stats snapshots: %w", loadErr)
	p.recordError(initialErr)
	if len(memberIDs) == 1 {
		p.restoreStatsMembers(memberIDs)
		return initialErr
	}

	errs := []error{initialErr}
	for _, memberID := range memberIDs {
		memberResult, err := p.source.LoadRealtimeStats(ctx, []int64{memberID})
		if err != nil {
			memberErr := fmt.Errorf("load stats snapshot for member %d: %w", memberID, err)
			p.restoreStatsMembers([]int64{memberID})
			p.recordError(memberErr)
			errs = append(errs, memberErr)
			continue
		}
		if err := p.publishStatsMember(ctx, memberID, memberResult[memberID]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishStatsMembers(ctx context.Context, memberIDs []int64, result map[int64]schemes.CloudCenterStats) error {
	var errs []error
	for _, memberID := range memberIDs {
		if err := p.publishStatsMember(ctx, memberID, result[memberID]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) publishStatsMember(ctx context.Context, memberID int64, stats schemes.CloudCenterStats) error {
	subject, err := StatsSubject(p.cfg.SubjectPrefix, memberID)
	if err != nil {
		return p.failStatsMember(memberID, fmt.Errorf("stats subject for member %d: %w", memberID, err))
	}
	message := StatsSnapshotMessage{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
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
		return p.failStatsMember(memberID, fmt.Errorf("publish stats snapshot for member %d: %w", memberID, err))
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
	for key := range p.schemeDirty {
		keys = append(keys, key)
		p.schemeInFlight[key] = struct{}{}
	}
	p.schemeDirty = make(map[schemeKey]struct{})
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
	for memberID := range p.statsDirty {
		memberIDs = append(memberIDs, memberID)
		p.statsInFlight[memberID] = struct{}{}
	}
	p.statsDirty = make(map[int64]struct{})
	sort.Slice(memberIDs, func(i, j int) bool { return memberIDs[i] < memberIDs[j] })
	return memberIDs
}

func (p *Publisher) restoreSchemeKeys(keys []schemeKey) {
	p.mu.Lock()
	for _, key := range keys {
		delete(p.schemeInFlight, key)
		p.schemeDirty[key] = struct{}{}
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
		delete(p.statsInFlight, memberID)
		p.statsDirty[memberID] = struct{}{}
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

func nonNilInstances(items []schemes.Instance) []schemes.Instance {
	if items == nil {
		return []schemes.Instance{}
	}
	return items
}

func nonNilStrings(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}
