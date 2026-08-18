package cloudrealtime

import "time"

type Diagnostics struct {
	SchemeQueueSize      int           `json:"schemeQueueSize"`
	StatsQueueSize       int           `json:"statsQueueSize"`
	AcceptedSchemeMarks  uint64        `json:"acceptedSchemeMarks"`
	AcceptedStatsMarks   uint64        `json:"acceptedStatsMarks"`
	CoalescedSchemeMarks uint64        `json:"coalescedSchemeMarks"`
	CoalescedStatsMarks  uint64        `json:"coalescedStatsMarks"`
	DroppedSchemeMarks   uint64        `json:"droppedSchemeMarks"`
	DroppedStatsMarks    uint64        `json:"droppedStatsMarks"`
	SchemePublishes      uint64        `json:"schemePublishes"`
	StatsPublishes       uint64        `json:"statsPublishes"`
	Errors               uint64        `json:"errors"`
	LastSuccess          time.Time     `json:"lastSuccess,omitempty"`
	LastError            string        `json:"lastError,omitempty"`
	LastPublishLatency   time.Duration `json:"lastPublishLatency"`
}
