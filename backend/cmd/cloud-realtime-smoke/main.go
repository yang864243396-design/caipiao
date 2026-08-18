package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
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

type runDependencies struct {
	stdout io.Writer
	stderr io.Writer
	getenv func(string) string
	newBus func(realtimebus.NATSConfig) (realtimebus.Bus, error)
}

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, os.Args[1:], runDependencies{
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
		newBus: func(config realtimebus.NATSConfig) (realtimebus.Bus, error) {
			return realtimebus.NewNATS(config)
		},
	})
}

func run(ctx context.Context, args []string, dependencies runDependencies) int {
	dependencies = normalizeRunDependencies(dependencies)
	flags := flag.NewFlagSet("cloud-realtime-smoke", flag.ContinueOnError)
	flags.SetOutput(dependencies.stderr)
	natsURLFlag := flags.String("nats", "", "NATS server or cluster URL (default: NATS_URL or nats://127.0.0.1:4222)")
	prefixFlag := flags.String("prefix", "", "NATS subject prefix (default: NATS_SUBJECT_PREFIX or caipiao)")
	memberID := flags.Int64("member-id", 0, "numeric member ID to observe (required)")
	timeout := flags.Duration("timeout", defaultTimeout, "maximum time to wait for both snapshot subjects")
	publish := flags.Bool("publish", false, "publish synthetic scheme and stats snapshots (explicitly mutates the subjects)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *memberID <= 0 {
		fmt.Fprintln(dependencies.stderr, "error: -member-id must be a positive numeric member ID")
		return 2
	}
	if *timeout <= 0 {
		fmt.Fprintln(dependencies.stderr, "error: -timeout must be positive")
		return 2
	}
	if err := ctx.Err(); err != nil {
		fmt.Fprintf(dependencies.stderr, "canceled: %v\n", err)
		return 130
	}

	natsURL := strings.TrimSpace(*natsURLFlag)
	if natsURL == "" {
		natsURL = envOrDefault(dependencies.getenv, "NATS_URL", "nats://127.0.0.1:4222")
	}
	prefix := strings.TrimSpace(*prefixFlag)
	if prefix == "" {
		prefix = envOrDefault(dependencies.getenv, "NATS_SUBJECT_PREFIX", "caipiao")
	}

	schemeSubject, err := cloudrealtime.SchemeSubject(prefix, *memberID)
	if err != nil {
		fmt.Fprintf(dependencies.stderr, "error: invalid scheme subject: %v\n", err)
		return 2
	}
	statsSubject, err := cloudrealtime.StatsSubject(prefix, *memberID)
	if err != nil {
		fmt.Fprintf(dependencies.stderr, "error: invalid stats subject: %v\n", err)
		return 2
	}

	bus, err := dependencies.newBus(realtimebus.NATSConfig{
		URL:             natsURL,
		Name:            "caipiao-cloud-realtime-smoke",
		User:            dependencies.getenv("NATS_USER"),
		Password:        dependencies.getenv("NATS_PASSWORD"),
		Token:           dependencies.getenv("NATS_TOKEN"),
		CredentialsFile: dependencies.getenv("NATS_CREDENTIALS_FILE"),
		ReconnectWait:   time.Second,
	})
	if err != nil {
		fmt.Fprintf(dependencies.stderr, "error: connect to NATS: %v\n", err)
		return 1
	}
	defer func() { _ = bus.Close() }()

	deadline := time.Now().Add(*timeout)
	if !waitConnected(ctx, bus, deadline) {
		if err := ctx.Err(); err != nil {
			fmt.Fprintf(dependencies.stderr, "canceled: %v\n", err)
			return 130
		}
		fmt.Fprintln(dependencies.stderr, "timeout: NATS did not become connected")
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
		fmt.Fprintf(dependencies.stderr, "error: subscribe scheme subject: %v\n", err)
		return 1
	}
	defer func() { _ = schemeSubscription.Unsubscribe() }()
	statsSubscription, err := subscribe(statsSubject, "stats")
	if err != nil {
		fmt.Fprintf(dependencies.stderr, "error: subscribe stats subject: %v\n", err)
		return 1
	}
	defer func() { _ = statsSubscription.Unsubscribe() }()

	if *publish {
		if err := publishSyntheticSnapshots(ctx, bus, schemeSubject, statsSubject); err != nil {
			if ctx.Err() != nil {
				fmt.Fprintf(dependencies.stderr, "canceled: %v\n", ctx.Err())
				return 130
			}
			fmt.Fprintf(dependencies.stderr, "error: publish synthetic snapshots: %v\n", err)
			return 1
		}
		fmt.Fprintln(dependencies.stdout, "synthetic_publish=enabled events=2")
	} else {
		fmt.Fprintln(dependencies.stdout, "synthetic_publish=disabled mode=read-only")
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
			fmt.Fprintf(dependencies.stdout, "event=%s schemaVersion=%d count=%d\n", event.kind, event.schemaVersion, counts[event.kind])
		case <-ctx.Done():
			fmt.Fprintf(dependencies.stderr, "canceled: %v\n", ctx.Err())
			return 130
		case <-timer.C:
			fmt.Fprintf(dependencies.stderr, "timeout: scheme_events=%d stats_events=%d dropped=%d\n", counts["scheme"], counts["stats"], dropped.Load())
			return 1
		}
	}

	fmt.Fprintf(dependencies.stdout, "PASS scheme_events=%d scheme_schemaVersion=%d stats_events=%d stats_schemaVersion=%d dropped=%d\n",
		counts["scheme"], versions["scheme"], counts["stats"], versions["stats"], dropped.Load())
	return 0
}

func normalizeRunDependencies(dependencies runDependencies) runDependencies {
	if dependencies.stdout == nil {
		dependencies.stdout = io.Discard
	}
	if dependencies.stderr == nil {
		dependencies.stderr = io.Discard
	}
	if dependencies.getenv == nil {
		dependencies.getenv = os.Getenv
	}
	if dependencies.newBus == nil {
		dependencies.newBus = func(config realtimebus.NATSConfig) (realtimebus.Bus, error) {
			return realtimebus.NewNATS(config)
		}
	}
	return dependencies
}

func publishSyntheticSnapshots(ctx context.Context, bus realtimebus.Bus, schemeSubject, statsSubject string) error {
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
	if err := bus.Publish(ctx, schemeSubject, schemePayload); err != nil {
		return err
	}
	return bus.Publish(ctx, statsSubject, statsPayload)
}

func waitConnected(ctx context.Context, bus realtimebus.Bus, deadline time.Time) bool {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for {
		if bus.Diagnostics().Connected {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			return false
		case <-ticker.C:
		}
	}
}

func envOrDefault(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
}
