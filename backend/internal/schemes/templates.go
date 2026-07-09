package schemes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrTemplateNotFound   = errors.New("scheme template not found")
	ErrInvalidTemplate    = errors.New("invalid scheme template")
	ErrTemplateForbidden  = errors.New("scheme template forbidden")
)

// ClientDraftDefinitionID 与客户端 session 草稿路由 param 一致（未落库方案）。
const ClientDraftDefinitionID = "new"

type TemplateRow struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	LotteryCode  string                 `json:"lotteryCode"`
	LotteryLabel string                 `json:"lotteryLabel"`
	Brief        string                 `json:"brief,omitempty"`
	SortOrder    int                    `json:"sortOrder"`
	Enabled      bool                   `json:"enabled"`
	MemberOwned  bool                   `json:"memberOwned,omitempty"`
	DefinitionID string                 `json:"definitionId,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

type SaveTemplateInput struct {
	ID          string
	Name        string
	LotteryCode string
	Brief       string
	SortOrder   int
	Enabled     bool
	Rounds      json.RawMessage
}

type defaultTemplateSeed struct {
	ID          string
	Name        string
	LotteryCode string
	Brief       string
	SortOrder   int
}

var defaultTemplateSeeds = []defaultTemplateSeed{
	{ID: "scheme_demo_1001", Name: "两期中跟挂停（附录演示）", LotteryCode: "tron_ffc_1m", Brief: "平台预置演示模板", SortOrder: 10},
	{ID: "tpl_demo_wave_3", Name: "三期推波方案", LotteryCode: "hash_ffc_1m", Brief: "三期推波结构示例", SortOrder: 20},
	{ID: "tpl_demo_plan_4", Name: "四期倍投计划", LotteryCode: "tron_ffc_1m", Brief: "四期计划表示例", SortOrder: 30},
	{ID: "tpl_demo_plan_6", Name: "六期倍投方案", LotteryCode: "eth_ffc_1m", Brief: "六期倍投表示例", SortOrder: 40},
}

func (s *Service) AdminListTemplates(ctx context.Context) ([]TemplateRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListSchemeTemplatesAdmin(ctx)
	if err != nil {
		return nil, err
	}
	return mapTemplateRows(rows), nil
}

type AdminTemplateListQuery struct {
	Page     int
	PageSize int
	Name     string
}

type AdminTemplateListResult struct {
	Items    []TemplateRow `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

func (s *Service) AdminListTemplatesPaged(ctx context.Context, q AdminTemplateListQuery) (AdminTemplateListResult, error) {
	if s == nil || s.q == nil {
		return AdminTemplateListResult{}, ErrUnavailable
	}
	page := q.Page
	pageSize := q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	nameKeyword := strings.TrimSpace(q.Name)
	total, err := s.q.CountSchemeTemplatesAdminPlatform(ctx, nameKeyword)
	if err != nil {
		return AdminTemplateListResult{}, err
	}
	offset := (page - 1) * pageSize
	rows, err := s.q.ListSchemeTemplatesAdminPlatformPaged(ctx, sqlcdb.ListSchemeTemplatesAdminPlatformPagedParams{
		PageLimit:   int32(pageSize),
		PageOffset:  int32(offset),
		NameKeyword: nameKeyword,
	})
	if err != nil {
		return AdminTemplateListResult{}, err
	}
	items := make([]TemplateRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapTemplateAdminPlatformPagedRow(row))
	}
	return AdminTemplateListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ClientListTemplates(ctx context.Context, account, definitionID string) ([]TemplateRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, member.ErrNotFound
		}
		return nil, err
	}
	definitionID = strings.TrimSpace(definitionID)
	if definitionID == "" {
		return nil, ErrInvalidTemplate
	}
	if definitionID == ClientDraftDefinitionID {
		rows, err := s.q.ListSchemeTemplatesPlatformEnabled(ctx)
		if err != nil {
			return nil, err
		}
		return mapTemplatePlatformEnabledRows(rows), nil
	}
	if _, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDefinitionNotFound
		}
		return nil, err
	}
	rows, err := s.q.ListSchemeTemplatesForDefinition(ctx, sqlcdb.ListSchemeTemplatesForDefinitionParams{
		DefinitionID: definitionID,
		MemberID:     m.ID,
	})
	if err != nil {
		return nil, err
	}
	return mapTemplateRowsForDefinition(rows), nil
}

func (s *Service) AdminGetTemplate(ctx context.Context, id string) (TemplateRow, error) {
	if s == nil || s.q == nil {
		return TemplateRow{}, ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return TemplateRow{}, ErrTemplateNotFound
	}
	row, err := s.q.GetSchemeTemplateByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRow{}, ErrTemplateNotFound
		}
		return TemplateRow{}, err
	}
	if row.MemberID.Valid || row.DefinitionID.Valid {
		return TemplateRow{}, ErrTemplateNotFound
	}
	return mapTemplateDetailRow(row), nil
}

func (s *Service) AdminSaveTemplate(ctx context.Context, in SaveTemplateInput) (TemplateRow, error) {
	if s == nil || s.q == nil {
		return TemplateRow{}, ErrUnavailable
	}
	in = normalizeTemplateInput(in)
	if in.Name == "" {
		return TemplateRow{}, ErrInvalidTemplate
	}
	var existingCfg []byte
	isPlatformUpdate := false
	if in.ID != "" {
		existing, err := s.q.GetSchemeTemplateByID(ctx, in.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TemplateRow{}, ErrTemplateNotFound
			}
			return TemplateRow{}, err
		}
		if existing.MemberID.Valid || existing.DefinitionID.Valid {
			return TemplateRow{}, ErrTemplateForbidden
		}
		isPlatformUpdate = true
		if in.LotteryCode == "" {
			in.LotteryCode = existing.LotteryCode
		}
		existingCfg = existing.Config
	}
	if in.LotteryCode == "" {
		in.LotteryCode = "tron_ffc_1m"
	}
	cfg, err := templateConfigWithRounds(existingCfg, in.Rounds)
	if err != nil {
		return TemplateRow{}, ErrInvalidTemplate
	}
	label, err := s.lotteryLabel(ctx, in.LotteryCode)
	if err != nil {
		return TemplateRow{}, err
	}
	if isPlatformUpdate {
		row, err := s.q.UpdateSchemeTemplatePlatform(ctx, sqlcdb.UpdateSchemeTemplatePlatformParams{
			ID:        in.ID,
			Name:      in.Name,
			Brief:     pgtype.Text{String: in.Brief, Valid: in.Brief != ""},
			SortOrder: int32(in.SortOrder),
			Enabled:   in.Enabled,
			Config:    cfg,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TemplateRow{}, ErrTemplateNotFound
			}
			return TemplateRow{}, err
		}
		return mapUpsertedTemplate(row, label), nil
	}
	if in.ID == "" {
		in.ID = newTemplateID()
	}
	row, err := s.q.UpsertSchemeTemplate(ctx, sqlcdb.UpsertSchemeTemplateParams{
		ID:           in.ID,
		Name:         in.Name,
		LotteryCode:  in.LotteryCode,
		Brief:        pgtype.Text{String: in.Brief, Valid: in.Brief != ""},
		SortOrder:    int32(in.SortOrder),
		Enabled:      in.Enabled,
		Config:       cfg,
		MemberID:     pgtype.Int8{},
		DefinitionID: pgtype.Text{},
	})
	if err != nil {
		return TemplateRow{}, err
	}
	return mapUpsertedTemplate(row, label), nil
}

func (s *Service) AdminDeleteTemplate(ctx context.Context, id string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrTemplateNotFound
	}
	n, err := s.q.DeleteSchemeTemplate(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *Service) AdminResetTemplates(ctx context.Context) ([]TemplateRow, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return nil, ErrUnavailable
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if err := q.DeleteAllSchemeTemplates(ctx); err != nil {
		return nil, err
	}
	for _, seed := range defaultTemplateSeeds {
		cfg, err := defaultTemplateSeedConfig()
		if err != nil {
			return nil, err
		}
		_, err = q.UpsertSchemeTemplate(ctx, sqlcdb.UpsertSchemeTemplateParams{
			ID:           seed.ID,
			Name:         seed.Name,
			LotteryCode:  seed.LotteryCode,
			Brief:        pgtype.Text{String: seed.Brief, Valid: seed.Brief != ""},
			SortOrder:    int32(seed.SortOrder),
			Enabled:      true,
			Config:       cfg,
			MemberID:     pgtype.Int8{},
			DefinitionID: pgtype.Text{},
		})
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.AdminListTemplates(ctx)
}

func normalizeTemplateInput(in SaveTemplateInput) SaveTemplateInput {
	sortOrder := in.SortOrder
	if sortOrder < 0 {
		sortOrder = 0
	}
	if sortOrder > 9999 {
		sortOrder = 9999
	}
	return SaveTemplateInput{
		ID:          strings.TrimSpace(in.ID),
		Name:        strings.TrimSpace(in.Name),
		LotteryCode: strings.TrimSpace(in.LotteryCode),
		Brief:       strings.TrimSpace(in.Brief),
		SortOrder:   sortOrder,
		Enabled:     in.Enabled,
		Rounds:      in.Rounds,
	}
}

func newTemplateID() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("tpl_%d%s", time.Now().UnixMilli(), hex.EncodeToString(b[:]))
}

func (s *Service) lotteryLabel(ctx context.Context, code string) (string, error) {
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return code, nil
		}
		return "", err
	}
	return row.DisplayName, nil
}

func mapTemplateRows(rows []sqlcdb.ListSchemeTemplatesAdminRow) []TemplateRow {
	out := make([]TemplateRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTemplateAdminRow(row))
	}
	return out
}

func mapTemplateRowsForDefinition(rows []sqlcdb.ListSchemeTemplatesForDefinitionRow) []TemplateRow {
	out := make([]TemplateRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTemplateDefinitionListRow(row))
	}
	return out
}

func mapTemplatePlatformEnabledRows(rows []sqlcdb.ListSchemeTemplatesPlatformEnabledRow) []TemplateRow {
	out := make([]TemplateRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTemplatePlatformEnabledRow(row))
	}
	return out
}

func mapTemplatePlatformEnabledRow(row sqlcdb.ListSchemeTemplatesPlatformEnabledRow) TemplateRow {
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Brief:        strings.TrimSpace(row.Brief),
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func mapTemplateDefinitionListRow(row sqlcdb.ListSchemeTemplatesForDefinitionRow) TemplateRow {
	defID := ""
	if row.DefinitionID.Valid {
		defID = strings.TrimSpace(row.DefinitionID.String)
	}
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Brief:        strings.TrimSpace(row.Brief),
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		MemberOwned:  defID != "",
		DefinitionID: defID,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func mapTemplateAdminRow(row sqlcdb.ListSchemeTemplatesAdminRow) TemplateRow {
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Brief:        strings.TrimSpace(row.Brief),
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func mapTemplateAdminPlatformPagedRow(row sqlcdb.ListSchemeTemplatesAdminPlatformPagedRow) TemplateRow {
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Brief:        strings.TrimSpace(row.Brief),
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func defaultTemplateSeedConfig() ([]byte, error) {
	return encodeTemplateConfig(map[string]interface{}{
		"rounds": []map[string]interface{}{
			{"mult": 0, "afterHit": 2, "afterMiss": 1},
			{"mult": 1, "afterHit": 2, "afterMiss": 3},
			{"mult": 3, "afterHit": 2, "afterMiss": 1},
		},
	})
}

func mapUpsertedTemplate(row sqlcdb.SchemeTemplate, label string) TemplateRow {
	brief := ""
	if row.Brief.Valid {
		brief = strings.TrimSpace(row.Brief.String)
	}
	defID := ""
	if row.DefinitionID.Valid {
		defID = strings.TrimSpace(row.DefinitionID.String)
	}
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: label,
		Brief:        brief,
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		MemberOwned:  defID != "",
		DefinitionID: defID,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func emptyTemplateConfig() []byte {
	return []byte("{}")
}

func parseTemplateConfig(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func encodeTemplateConfig(cfg map[string]interface{}) ([]byte, error) {
	if cfg == nil {
		return emptyTemplateConfig(), nil
	}
	return json.Marshal(cfg)
}

func mapTemplateDetailRow(row sqlcdb.GetSchemeTemplateByIDRow) TemplateRow {
	defID := ""
	if row.DefinitionID.Valid {
		defID = strings.TrimSpace(row.DefinitionID.String)
	}
	return TemplateRow{
		ID:           row.ID,
		Name:         row.Name,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Brief:        strings.TrimSpace(row.Brief),
		SortOrder:    int(row.SortOrder),
		Enabled:      row.Enabled,
		MemberOwned:  defID != "",
		DefinitionID: defID,
		Config:       parseTemplateConfig(row.Config),
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}
