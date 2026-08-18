package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"caipiao/backend/internal/cloudrealtime"
	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/schemes"
)

const defaultTimeout = 15 * time.Second

type observedEvent struct {
	kind          string
	schemaVersion int
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("cloud-realtime-smoke", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	natsURL := flags.String("nats", envOrDefault("NATS_URL", "nats://127.0.0.1:4222"), "NATS server or cluster URL")
	prefix := flags.String("prefix", envOrDefault("NATS_SUBJECT_PREFIX", "caipiao"), "NATS subject prefix")
	memberID := flags.Int64("member-id", 0, "numeric member ID to observe (required)")
	timeout := flags.Duration("timeout", defaultTimeout, "maximum time to wait for both snapshot subjects")
	publish := flags.Bool("publish", false, "publish synthetic scheme and stats snapshots (explicitly mutates the subjects)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *memberID <= 0 {
		fmt.Fprintln(os.Stderr, "error: -member-id must be a positive numeric member ID")
		return 2
	}
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "error: -timeout must be positive")
		return 2
	}

	schemeSubject, err := cloudrealtime.SchemeSubject(*prefix, *memberID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid scheme subject: %v\n", err)
		return 2
	}
	statsSubject, err := cloudrealtime.StatsSubject(*prefix, *memberID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid stats subject: %v\n", err)
		return 2
	}

	bus, err := realtimebus.NewNATS(realtimebus.NATSConfig{
		URL:             strings.TrimSpace(*natsURL),
		Name:            "caipiao-cloud-realtime-smoke",
		User:            os.Getenv("NATS_USER"),
		Password:        os.Getenv("NATS_PASSWORD"),
		Token:           os.Getenv("NATS_TOKEN"),
		CredentialsFile: os.Getenv("NATS_CREDENTIALS_FILE"),
		ReconnectWait:   time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: connect to NATS: %v\n", err)
		return 1
	}
	defer func() { _ = bus.Close() }()

	deadline := time.Now().Add(*timeout)
	if !waitConnected(bus, deadline) {
		fmt.Fprintln(os.Stderr, "timeout: NATS did not become connected")
		return 1
	}

	events := make(chan observedEvent, 128)
	var dropped atomic.Uint64
	subscribe := func(subject, kind string) (realtimebus.Subscription, error) {
		return bus.Subscribe(subject, func(_ string, payload []byte) {
			var envelope struct {
				SchemaVersion int `json:"schemaVersion"`
			}
			if err := json.Unmarshal(payload, &envelope); err != nil {
				envelope.SchemaVersion = 0
			}
			select {
			case events <- observedEvent{kind: kind, schemaVersion: envelope.SchemaVersion}:
			default:
				dropped.Add(1)
			}
		})
	}

	schemeSubscription, err := subscribe(schemeSubject, "scheme")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: subscribe scheme subject: %v\n", err)
		return 1
	}
	defer func() { _ = schemeSubscription.Unsubscribe() }()
	statsSubscription, err := subscribe(statsSubject, "stats")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: subscribe stats subject: %v\n", err)
		return 1
	}
	defer func() { _ = statsSubscription.Unsubscribe() }()

	if *publish {
		if err := publishSyntheticSnapshots(bus, schemeSubject, statsSubject); err != nil {
			fmt.Fprintf(os.Stderr, "error: publish synthetic snapshots: %v\n", err)
			return 1
		}
		fmt.Println("synthetic_publish=enabled events=2")
	} else {
		fmt.Println("synthetic_publish=disabled mode=read-only")
	}

	counts := map[string]int{"scheme": 0, "stats": 0}
	versions := map[string]int{"scheme": 0, "stats": 0}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for counts["scheme"] == 0 || counts["stats"] == 0 {
		select {
		case event := <-events:
			counts[event.kind]++
			versions[event.kind] = event.schemaVersion
			fmt.Printf("event=%s schemaVersion=%d count=%d\n", event.kind, event.schemaVersion, counts[event.kind])
		case <-timer.C:
			fmt.Fprintf(os.Stderr, "timeout: scheme_events=%d stats_events=%d dropped=%d\n", counts["scheme"], counts["stats"], dropped.Load())
			return 1
		}
	}

	fmt.Printf("PASS scheme_events=%d scheme_schemaVersion=%d stats_events=%d stats_schemaVersion=%d dropped=%d\n",
		counts["scheme"], versions["scheme"], counts["stats"], versions["stats"], dropped.Load())
	return 0
}

func publishSyntheticSnapshots(bus realtimebus.Bus, schemeSubject, statsSubject string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	schemePayload, err := json.Marshal(cloudrealtime.SchemeSnapshotMessage{
		SchemaVersion: cloudrealtime.SchemaVersion,
		GeneratedAt:   now,
		Items:         []schemes.Instance{},
		RemovedIDs:    []string{},
	})
	if err != nil {
		return err
	}
	statsPayload, err := json.Marshal(cloudrealtime.StatsSnapshotMessage{
		SchemaVersion: cloudrealtime.SchemaVersion,
		GeneratedAt:   now,
	})
	if err != nil {
		return err
	}
	if err := bus.Publish(context.Background(), schemeSubject, schemePayload); err != nil {
		return err
	}
	return bus.Publish(context.Background(), statsSubject, statsPayload)
}

func waitConnected(bus realtimebus.Bus, deadline time.Time) bool {
	for time.Now().Before(deadline) {
		if bus.Diagnostics().Connected {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
