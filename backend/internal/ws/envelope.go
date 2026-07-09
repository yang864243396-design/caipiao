package ws

import (
	"fmt"
	"time"
)

const (
	FrameTypeSystem  = "system"
	FrameTypeCommand = "command"
	FrameTypeEvent   = "event"
	FrameTypeError   = "error"
)

const (
	NameConnected  = "system.connected"
	NameAuthOK     = "system.auth.ok"
	NameSubscribed = "system.subscribed"
	NamePong       = "system.pong"
	NameError      = "system.error"

	NameMaintenanceChanged = "public.maintenance.changed"

	NameSchemeInstanceUpdated = "client.scheme.instance.updated"
	NameWalletUpdated         = "client.wallet.updated"

	NameWithdrawQueueChanged = "admin.withdraw.queue.changed"
	NameSchemeMonitorChanged   = "admin.scheme.monitor.changed"
	NameDashboardKpiChanged    = "admin.dashboard.kpi.changed"

	NameDrawResult = "public.draw.result"
)

const (
	TopicPublicMaintenance    = "public.maintenance"
	TopicClientSchemeInstance = "client.scheme.instance"
	TopicClientWallet         = "client.wallet"
	TopicAdminWithdrawQueue   = "admin.withdraw.queue"
	TopicAdminSchemeMonitor   = "admin.scheme.monitor"
	TopicAdminDashboardKpi    = "admin.dashboard.kpi"
)

func TopicPublicDraw(lotteryCode string) string {
	return "public.draw:" + lotteryCode
}

type Envelope struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Topic   string `json:"topic,omitempty"`
	EventID string `json:"eventId,omitempty"`
	TS      string `json:"ts"`
	Payload any    `json:"payload,omitempty"`
}

type MaintenanceChangedPayload struct {
	Enabled             bool   `json:"enabled"`
	Title               string `json:"title,omitempty"`
	Message             string `json:"message,omitempty"`
	PopupAnnouncementID string `json:"popupAnnouncementId,omitempty"`
	PopupAnnouncement   any    `json:"popupAnnouncement,omitempty"`
}

type SchemeInstancePayload struct {
	InstanceID string `json:"instanceId"`
	RunMode    string `json:"runMode"`
	SimBet     bool   `json:"simBet"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

type WalletUpdatedPayload struct {
	Available float64 `json:"available"`
	Frozen    float64 `json:"frozen"`
	Currency  string  `json:"currency"`
	Reason    string  `json:"reason,omitempty"`
}

type WithdrawQueueChangedPayload struct {
	OrderNo string `json:"orderNo"`
	Status  string `json:"status"`
	Action  string `json:"action"`
	Hint    string `json:"hint,omitempty"`
}

type AdminSchemeMonitorPayload struct {
	InstanceID string `json:"instanceId"`
	Status     string `json:"status"`
	Action     string `json:"action"`
	Hint       string `json:"hint,omitempty"`
}

type DashboardKpiChangedPayload struct {
	Metric  string  `json:"metric"`
	OrderNo string  `json:"orderNo,omitempty"`
	Amount  float64 `json:"amount,omitempty"`
	Action  string  `json:"action"`
	Hint    string  `json:"hint,omitempty"`
}

type DrawResultPayload struct {
	LotteryCode string   `json:"lotteryCode"`
	IssueNo     string   `json:"issueNo"`
	PeriodShort string   `json:"periodShort,omitempty"`
	Balls       []string `json:"balls"`
	SumValue    int      `json:"sumValue"`
	DrawnAt     string   `json:"drawnAt"`
	Hint        string   `json:"hint,omitempty"`
}

func NewEvent(name, topic string, payload any) Envelope {
	return Envelope{
		Type:    FrameTypeEvent,
		Name:    name,
		Topic:   topic,
		EventID: fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		TS:      time.Now().UTC().Format(time.RFC3339Nano),
		Payload: payload,
	}
}

func SystemFrame(name string, payload any) Envelope {
	return Envelope{
		Type:    FrameTypeSystem,
		Name:    name,
		TS:      time.Now().UTC().Format(time.RFC3339Nano),
		Payload: payload,
	}
}

func ErrorFrame(code int, message string) Envelope {
	return Envelope{
		Type: FrameTypeError,
		Name: NameError,
		TS:   time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}
