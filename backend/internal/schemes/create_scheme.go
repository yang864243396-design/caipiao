package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

var (
	ErrNameDuplicate        = errors.New("scheme name duplicate")
	ErrInvalidCreateRequest = errors.New("invalid create scheme request")
)

type CreateDefinitionInput struct {
	Kind         string
	SchemeName   string
	LotteryCode  string
	RunTypeID    string
	PlayTypeID   string
	SubPlayID    string
}

func (s *Service) CreateDefinition(ctx context.Context, account string, in CreateDefinitionInput) (Definition, error) {
	if s == nil || s.q == nil {
		return Definition{}, ErrUnavailable
	}
	normalizeCreateInput(&in)
	if err := validateCreateInput(in); err != nil {
		return Definition{}, err
	}
	if in.Kind == "custom" {
		if err := s.validateCreateRunTypePlay(ctx, in); err != nil {
			return Definition{}, err
		}
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Definition{}, member.ErrNotFound
		}
		return Definition{}, err
	}

	schemeName := in.SchemeName
	if schemeName == "" {
		schemeName = "新方案"
	}
	lotteryLabel := ""
	if in.LotteryCode != "" {
		lotteryLabel, err = s.lotteryLabel(ctx, in.LotteryCode)
		if err != nil {
			return Definition{}, err
		}
	}

	cfg, err := s.buildCreateDefinitionConfig(ctx, in, schemeName)
	if err != nil {
		return Definition{}, err
	}

	defID := fmt.Sprintf("def-%d-%d", m.ID, time.Now().UnixMilli())
	row, err := s.q.InsertSchemeDefinition(ctx, sqlcdb.InsertSchemeDefinitionParams{
		ID:                defID,
		MemberID:          m.ID,
		Kind:              in.Kind,
		SchemeName:        schemeName,
		LotteryCode:       in.LotteryCode,
		LotteryLabel:      lotteryLabel,
		ShareStatus:       "private",
		ShareStatusLocked: false,
		SourceSnapshotID:  pgtype.Text{},
		Config:            cfg,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Definition{}, ErrNameDuplicate
		}
		return Definition{}, err
	}
	return mapDefinitionRow(row, false), nil
}

func normalizeCreateInput(in *CreateDefinitionInput) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.SchemeName = strings.TrimSpace(in.SchemeName)
	in.LotteryCode = strings.TrimSpace(in.LotteryCode)
	in.RunTypeID = strings.TrimSpace(in.RunTypeID)
	in.PlayTypeID = strings.TrimSpace(in.PlayTypeID)
	in.SubPlayID = strings.TrimSpace(in.SubPlayID)
}

func (s *Service) buildCreateDefinitionConfig(
	ctx context.Context,
	in CreateDefinitionInput,
	schemeName string,
) ([]byte, error) {
	cfg := map[string]string{
		"schemeName":  schemeName,
		"lotteryCode": in.LotteryCode,
		"runTypeId":   in.RunTypeID,
		"playTypeId":  in.PlayTypeID,
		"subPlayId":   in.SubPlayID,
		"typeId":      in.PlayTypeID,
		"subId":       in.SubPlayID,
	}

	// 内置计画：创建时无彩种/玩法，待选择收藏方案后物化（v8 §3.6）
	if in.LotteryCode == "" {
		return json.Marshal(cfg)
	}

	cat, err := s.q.GetLotteryCatalogByCode(ctx, in.LotteryCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: lotteryCode 无效", ErrInvalidCreateRequest)
		}
		return nil, err
	}
	template := strings.TrimSpace(cat.PlayTemplate.String)
	if cat.PlayTemplate.Valid && template != "" {
		cfg["playTemplate"] = template
		sub, subErr := s.q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
			TemplateCode: template,
			TypeID:       in.PlayTypeID,
			SubID:        in.SubPlayID,
		})
		if subErr != nil {
			if errors.Is(subErr, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%w: 玩法不存在", ErrInvalidCreateRequest)
			}
			return nil, subErr
		}
		if betMode := strings.TrimSpace(sub.BetMode.String); sub.BetMode.Valid && betMode != "" {
			cfg["betMode"] = betMode
		}
	}
	return json.Marshal(cfg)
}

func validateCreateInput(in CreateDefinitionInput) error {
	if in.Kind != "custom" && in.Kind != "contrary" && in.Kind != "follow" {
		return fmt.Errorf("%w: kind 无效", ErrInvalidCreateRequest)
	}
	if in.SchemeName != "" && len([]rune(in.SchemeName)) > 128 {
		return fmt.Errorf("%w: schemeName 过长", ErrInvalidCreateRequest)
	}
	if in.RunTypeID == "" {
		return fmt.Errorf("%w: runTypeId 不能为空", ErrInvalidCreateRequest)
	}
	// 内置计画：彩种与玩法随收藏方案物化带出，创建时放宽（v8 §2）
	if in.Kind == "custom" && in.RunTypeID == RunTypeBuiltinPlan {
		return nil
	}
	if in.LotteryCode == "" {
		return fmt.Errorf("%w: lotteryCode 不能为空", ErrInvalidCreateRequest)
	}
	if in.PlayTypeID == "" || in.SubPlayID == "" {
		return fmt.Errorf("%w: playTypeId / subPlayId 不能为空", ErrInvalidCreateRequest)
	}
	return nil
}

func (s *Service) validateCreateRunTypePlay(ctx context.Context, in CreateDefinitionInput) error {
	if in.RunTypeID == RunTypeBuiltinPlan || in.LotteryCode == "" {
		return nil
	}
	cat, err := s.q.GetLotteryCatalogByCode(ctx, in.LotteryCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: lotteryCode 无效", ErrInvalidCreateRequest)
		}
		return err
	}
	if !cat.PlayTemplate.Valid || strings.TrimSpace(cat.PlayTemplate.String) == "" {
		return ValidateRunTypePlay(NormalizeRunTypeID(in.RunTypeID), in.PlayTypeID, in.SubPlayID, "", "")
	}
	sub, err := s.q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
		TemplateCode: strings.TrimSpace(cat.PlayTemplate.String),
		TypeID:       in.PlayTypeID,
		SubID:        in.SubPlayID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: 玩法不存在", ErrInvalidCreateRequest)
		}
		return err
	}
	template := strings.TrimSpace(cat.PlayTemplate.String)
	group := guajiGroupFromSegment(sub.SegmentRule)
	if group == "" {
		group = s.playTypeLabel(ctx, template, in.PlayTypeID)
	}
	return ValidateRunTypePlay(NormalizeRunTypeID(in.RunTypeID), in.PlayTypeID, in.SubPlayID, group, sub.Label)
}

func (s *Service) playTypeLabel(ctx context.Context, templateCode, typeID string) string {
	types, err := s.q.ListPlayTypesByTemplate(ctx, templateCode)
	if err != nil {
		return ""
	}
	for _, pt := range types {
		if pt.TypeID == typeID {
			return strings.TrimSpace(pt.Label)
		}
	}
	return ""
}
