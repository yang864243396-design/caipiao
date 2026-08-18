package cloudrealtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"caipiao/backend/internal/schemes"
)

const SchemaVersion = 1

type SchemeSnapshotMessage struct {
	SchemaVersion int                `json:"schemaVersion"`
	GeneratedAt   string             `json:"generatedAt"`
	Items         []schemes.Instance `json:"items"`
	RemovedIDs    []string           `json:"removedIds"`
}

type StatsSnapshotMessage struct {
	SchemaVersion int                      `json:"schemaVersion"`
	GeneratedAt   string                   `json:"generatedAt"`
	Stats         schemes.CloudCenterStats `json:"stats"`
}

type SnapshotSource interface {
	LoadRealtimeSchemeSnapshots(context.Context, []schemes.RealtimeInstanceRef) (schemes.RealtimeSchemeSnapshotResult, error)
	LoadRealtimeStats(context.Context, []int64) (map[int64]schemes.CloudCenterStats, error)
}

func SchemeSubject(prefix string, memberID int64) (string, error) {
	return memberSubject(prefix, memberID, "scheme")
}

func StatsSubject(prefix string, memberID int64) (string, error) {
	return memberSubject(prefix, memberID, "cloud_stats")
}

func memberSubject(prefix string, memberID int64, suffix string) (string, error) {
	prefix = strings.Trim(prefix, ".")
	if prefix == "" {
		return "", errors.New("subject prefix is required")
	}
	if memberID <= 0 {
		return "", errors.New("member ID must be positive")
	}
	if strings.Contains(prefix, "..") {
		return "", errors.New("subject prefix contains an empty token")
	}
	for _, char := range prefix {
		if char == '*' || char == '>' || unicode.IsSpace(char) {
			return "", errors.New("subject prefix contains an unsafe character")
		}
	}
	return fmt.Sprintf("%s.client.%d.%s", prefix, memberID, suffix), nil
}
