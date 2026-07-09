package instances

import (
	"errors"
	"sync"
	"time"
)

type Status string

const (
	StatusPending     Status = "pending"
	StatusRunning     Status = "running"
	StatusPaused      Status = "paused"
	StatusSoftStopped Status = "soft_stopped"
)

type Kind string

const (
	KindCustom   Kind = "custom"
	KindContrary Kind = "contrary"
	KindFollow   Kind = "follow"
)

type RunMode string

const (
	RunModeReal RunMode = "real"
	RunModeSim  RunMode = "sim"
)

var (
	ErrNotFound      = errors.New("instance not found")
	ErrInvalidAction = errors.New("invalid action for current status")
)

type Instance struct {
	ID           string    `json:"id"`
	DefinitionID string    `json:"definitionId"`
	Kind         Kind      `json:"kind"`
	SchemeName   string    `json:"schemeName"`
	LotteryCode  string    `json:"lotteryCode"`
	LotteryLabel string    `json:"lotteryLabel"`
	Status       Status    `json:"status"`
	StatusLabel  string    `json:"statusLabel"`
	RunMode      RunMode   `json:"runMode"`
	Turnover     float64   `json:"turnover"`
	PnL          float64   `json:"pnl"`
	RunTimeSec   int       `json:"runTimeSec"`
	LookbackPnL  float64   `json:"lookbackPnl"`
	Multiplier   float64   `json:"multiplier"`
	CountdownSec int       `json:"countdownSec"`
	SimBet       bool      `json:"simBet"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Store struct {
	mu   sync.RWMutex
	list []Instance
}

func NewStore() *Store {
	now := time.Now().UTC()
	return &Store{
		list: []Instance{
			{
				ID: "cloud_demo_2001", DefinitionID: "custom_demo_4001", Kind: KindCustom,
				SchemeName: "漠北万位", LotteryCode: "eth_ffc_1m", LotteryLabel: "以太坊1分彩",
				Status: StatusPending, StatusLabel: statusLabel(StatusPending),
				RunMode: RunModeReal, Turnover: 0, PnL: 0, RunTimeSec: 0, LookbackPnL: 0,
				Multiplier: 1, CountdownSec: 7, SimBet: false,
				CreatedAt: now, UpdatedAt: now,
			},
			{
				ID: "inst-s2", DefinitionID: "def-s2", Kind: KindCustom,
				SchemeName: "刚好", LotteryCode: "tron_ffc_1m", LotteryLabel: "波场1分彩",
				Status: StatusPending, StatusLabel: statusLabel(StatusPending),
				RunMode: RunModeSim, Turnover: 0, PnL: 0, RunTimeSec: 0, LookbackPnL: 0,
				Multiplier: 1, CountdownSec: 7, SimBet: true,
				CreatedAt: now, UpdatedAt: now,
			},
		},
	}
}

func statusLabel(s Status) string {
	switch s {
	case StatusRunning:
		return "运行中"
	case StatusPaused:
		return "已暂停"
	case StatusSoftStopped:
		return "已封停"
	default:
		return "待开启"
	}
}

func (s *Store) List() []Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Instance, len(s.list))
	copy(out, s.list)
	return out
}

func (s *Store) Get(id string) (Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.list {
		if row.ID == id {
			return row, true
		}
	}
	return Instance{}, false
}

func (s *Store) Start(id string) (Instance, error) {
	return s.transition(id, func(row *Instance) error {
		if row.Status != StatusPending {
			return ErrInvalidAction
		}
		row.Status = StatusRunning
		row.StatusLabel = statusLabel(StatusRunning)
		return nil
	})
}

func (s *Store) Pause(id string) (Instance, error) {
	return s.transition(id, func(row *Instance) error {
		if row.Status != StatusRunning {
			return ErrInvalidAction
		}
		row.Status = StatusPaused
		row.StatusLabel = statusLabel(StatusPaused)
		return nil
	})
}

func (s *Store) Resume(id string) (Instance, error) {
	return s.transition(id, func(row *Instance) error {
		if row.Status != StatusPaused {
			return ErrInvalidAction
		}
		row.Status = StatusRunning
		row.StatusLabel = statusLabel(StatusRunning)
		return nil
	})
}

func (s *Store) UpdateMultiplier(id string, multiplier float64) (Instance, error) {
	if multiplier < 1 || float64(int(multiplier)) != multiplier {
		return Instance{}, ErrInvalidAction
	}
	return s.transition(id, func(row *Instance) error {
		row.Multiplier = multiplier
		return nil
	})
}

func (s *Store) transition(id string, fn func(*Instance) error) (Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.list {
		if s.list[i].ID != id {
			continue
		}
		if err := fn(&s.list[i]); err != nil {
			return Instance{}, err
		}
		s.list[i].UpdatedAt = time.Now().UTC()
		return s.list[i], nil
	}
	return Instance{}, ErrNotFound
}
