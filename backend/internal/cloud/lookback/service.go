package lookback



import (

	"context"

	"errors"

	"fmt"



	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgtype"



	"caipiao/backend/internal/db"

	"caipiao/backend/internal/db/sqlcdb"

)



var (

	ErrUnavailable     = errors.New("lookback service unavailable")

	ErrInvalidSettings = errors.New("invalid lookback settings")

)



type Service struct {

	q   *sqlcdb.Queries

	mem *Store

}



func NewService(pool *db.Pool) *Service {

	s := &Service{mem: NewStore()}

	if pool != nil {

		s.q = sqlcdb.New(pool)

	}

	return s

}



func (s *Service) hasDB() bool {

	return s != nil && s.q != nil

}



func (s *Service) GetMemory() Settings {

	if s == nil || s.mem == nil {

		return defaultSettings()

	}

	return s.mem.Get()

}



func (s *Service) Get(ctx context.Context, memberID int64) (Settings, error) {

	if !s.hasDB() {

		return s.GetMemory(), nil

	}

	row, err := s.q.GetMemberLookbackSettings(ctx, memberID)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {

			return defaultSettings(), nil

		}

		return Settings{}, err

	}

	return mapSettingsRow(row), nil

}



func (s *Service) Put(ctx context.Context, memberID int64, next Settings) (Settings, error) {

	normalizeSettings(&next)

	if err := validateSettings(next); err != nil {

		return Settings{}, err

	}

	if !s.hasDB() {

		if s.mem == nil {

			return Settings{}, ErrUnavailable

		}

		return s.mem.Put(next), nil

	}

	row, err := s.q.UpsertMemberLookbackSettings(ctx, upsertParams(memberID, next))

	if err != nil {

		return Settings{}, err

	}

	return mapUpsertRow(row), nil

}



func defaultSettings() Settings {

	return Settings{

		Judgment:              JudgmentIndividual,

		SingleProfitThreshold: 100,

		SingleLossThreshold:   0,

	}

}



func normalizeSettings(s *Settings) {

	if s == nil {

		return

	}

	if !s.ApplyFormal && !s.ApplySim && len(s.RunModes) > 0 {

		SyncApplyFlagsFromRunModes(s)

	} else {

		SyncRunModesFromApplyFlags(s)

	}

}



func validateSettings(s Settings) error {

	if s.Judgment != JudgmentNone && s.Judgment != JudgmentIndividual && s.Judgment != JudgmentOverall {

		return fmt.Errorf("%w: judgment 须为空、individual 或 overall", ErrInvalidSettings)

	}

	return nil

}



func mapSettingsRow(row sqlcdb.GetMemberLookbackSettingsRow) Settings {

	return mapRowFields(row.ApplyFormal, row.ApplySim, row.RunMode, row.Judgment,

		row.SingleProfitThreshold, row.SingleLossThreshold,

		row.OverallProfitThreshold, row.OverallLossThreshold,

		row.SchemeWinsMin, row.SchemeWinsMax, row.PeriodProfit, row.PeriodLoss)

}



func mapUpsertRow(row sqlcdb.UpsertMemberLookbackSettingsRow) Settings {

	return mapRowFields(row.ApplyFormal, row.ApplySim, row.RunMode, row.Judgment,

		row.SingleProfitThreshold, row.SingleLossThreshold,

		row.OverallProfitThreshold, row.OverallLossThreshold,

		row.SchemeWinsMin, row.SchemeWinsMax, row.PeriodProfit, row.PeriodLoss)

}



func mapRowFields(

	applyFormal, applySim bool,

	runMode, judgment string,

	singleProfit, singleLoss, overallProfit, overallLoss,

	schemeWinsMin, schemeWinsMax, periodProfit, periodLoss pgtype.Numeric,

) Settings {

	s := Settings{

		ApplyFormal:            applyFormal,

		ApplySim:               applySim,

		Judgment:               Judgment(judgment),

		SingleProfitThreshold:  numericToFloat(singleProfit),

		SingleLossThreshold:    numericToFloat(singleLoss),

		OverallProfitThreshold: numericToFloat(overallProfit),

		OverallLossThreshold:   numericToFloat(overallLoss),

		SchemeWinsMin:          numericToFloat(schemeWinsMin),

		SchemeWinsMax:          numericToFloat(schemeWinsMax),

		PeriodProfit:           numericToFloat(periodProfit),

		PeriodLoss:             numericToFloat(periodLoss),

	}

	if !applyFormal && !applySim {

		s.RunModes = DecodeRunModes(runMode)

		SyncApplyFlagsFromRunModes(&s)

	} else {

		SyncRunModesFromApplyFlags(&s)

	}

	return s

}



func upsertParams(memberID int64, s Settings) sqlcdb.UpsertMemberLookbackSettingsParams {

	return sqlcdb.UpsertMemberLookbackSettingsParams{

		MemberID:               memberID,

		RunMode:                EncodeRunModes(s.RunModes),

		ApplyFormal:            s.ApplyFormal,

		ApplySim:               s.ApplySim,

		Judgment:               string(s.Judgment),

		SingleProfitThreshold:  floatToNumeric(s.SingleProfitThreshold),

		SingleLossThreshold:    floatToNumeric(s.SingleLossThreshold),

		OverallProfitThreshold: floatToNumeric(s.OverallProfitThreshold),

		OverallLossThreshold:   floatToNumeric(s.OverallLossThreshold),

		SchemeWinsMin:          floatToNumeric(s.SchemeWinsMin),

		SchemeWinsMax:          floatToNumeric(s.SchemeWinsMax),

		PeriodProfit:           floatToNumeric(s.PeriodProfit),

		PeriodLoss:             floatToNumeric(s.PeriodLoss),

	}

}



func floatToNumeric(v float64) pgtype.Numeric {

	var n pgtype.Numeric

	_ = n.Scan(fmt.Sprintf("%.2f", v))

	return n

}



func numericToFloat(n pgtype.Numeric) float64 {

	if !n.Valid {

		return 0

	}

	f, err := n.Float64Value()

	if err != nil || !f.Valid {

		return 0

	}

	return f.Float64

}

