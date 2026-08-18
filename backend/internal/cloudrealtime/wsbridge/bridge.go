package wsbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"caipiao/backend/internal/cloudrealtime"
	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/ws"
)

type Bridge struct {
	bus           realtimebus.Bus
	subjectPrefix string
}

func New(bus realtimebus.Bus, subjectPrefix string) *Bridge {
	return &Bridge{bus: bus, subjectPrefix: subjectPrefix}
}

func (b *Bridge) SubscribeMember(memberID int64, emit func(ws.Envelope)) (func(), error) {
	if b == nil || b.bus == nil {
		return nil, errors.New("realtime bus is required")
	}
	if emit == nil {
		return nil, errors.New("member event emitter is required")
	}
	schemeSubject, err := cloudrealtime.SchemeSubject(b.subjectPrefix, memberID)
	if err != nil {
		return nil, err
	}
	statsSubject, err := cloudrealtime.StatsSubject(b.subjectPrefix, memberID)
	if err != nil {
		return nil, err
	}

	schemeSubscription, err := b.bus.Subscribe(schemeSubject, func(subject string, payload []byte) {
		if subject != schemeSubject {
			return
		}
		var message cloudrealtime.SchemeSnapshotMessage
		if json.Unmarshal(payload, &message) != nil || !validMessage(message.SchemaVersion, message.GeneratedAt) {
			return
		}
		emit(ws.NewEvent(ws.NameSchemeInstancesSnapshot, ws.TopicClientSchemeInstance, message))
		for _, item := range message.Items {
			if item.ID == "" || item.Status == "" {
				continue
			}
			emit(legacyHint(item.ID, item.Status))
		}
		for _, instanceID := range message.RemovedIDs {
			if instanceID != "" {
				emit(legacyHint(instanceID, "stopped"))
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe scheme snapshots: %w", err)
	}
	statsSubscription, err := b.bus.Subscribe(statsSubject, func(subject string, payload []byte) {
		if subject != statsSubject {
			return
		}
		var message cloudrealtime.StatsSnapshotMessage
		if json.Unmarshal(payload, &message) != nil || !validMessage(message.SchemaVersion, message.GeneratedAt) {
			return
		}
		emit(ws.NewEvent(ws.NameCloudStatsSnapshot, ws.TopicClientCloudStats, message))
	})
	if err != nil {
		_ = schemeSubscription.Unsubscribe()
		return nil, fmt.Errorf("subscribe cloud stats snapshots: %w", err)
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			_ = schemeSubscription.Unsubscribe()
			_ = statsSubscription.Unsubscribe()
		})
	}, nil
}

func validMessage(schemaVersion int, generatedAt string) bool {
	if schemaVersion != cloudrealtime.SchemaVersion {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, generatedAt)
	return err == nil
}

func legacyHint(instanceID, status string) ws.Envelope {
	return ws.NewEvent(ws.NameSchemeInstanceUpdated, ws.TopicClientSchemeInstance, ws.SchemeInstanceHintPayload{
		InstanceID: instanceID,
		Status:     status,
		Hint:       "refresh_running_list",
	})
}

var _ ws.MemberEventSource = (*Bridge)(nil)
