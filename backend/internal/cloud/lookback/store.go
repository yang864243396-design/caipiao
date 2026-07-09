package lookback



import "sync"



type Judgment string



const (

	JudgmentNone       Judgment = ""

	JudgmentIndividual Judgment = "individual"

	JudgmentOverall    Judgment = "overall"

)



type Settings struct {

	ApplyFormal bool `json:"applyFormal"`

	ApplySim    bool `json:"applySim"`

	// RunModes 兼容旧客户端；读写时与 applyFormal/applySim 互相同步。

	RunModes               []RunMode `json:"runModes,omitempty"`

	Judgment               Judgment  `json:"judgment"`

	SingleProfitThreshold  float64   `json:"singleProfitThreshold"`

	SingleLossThreshold    float64   `json:"singleLossThreshold"`

	OverallProfitThreshold float64   `json:"overallProfitThreshold"`

	OverallLossThreshold   float64   `json:"overallLossThreshold"`

	SchemeWinsMin          float64   `json:"schemeWinsMin"`

	SchemeWinsMax          float64   `json:"schemeWinsMax"`

	PeriodProfit           float64   `json:"periodProfit"`

	PeriodLoss             float64   `json:"periodLoss"`

}



type Store struct {

	mu   sync.RWMutex

	data Settings

}



func NewStore() *Store {

	return &Store{

		data: Settings{

			Judgment:              JudgmentIndividual,

			SingleProfitThreshold: 100,

			SingleLossThreshold:   0,

		},

	}

}



func (s *Store) Get() Settings {

	s.mu.RLock()

	defer s.mu.RUnlock()

	return s.data

}



func (s *Store) Put(next Settings) Settings {

	s.mu.Lock()

	defer s.mu.Unlock()

	s.data = next

	return s.data

}

