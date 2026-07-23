package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

var (
	ErrDeleteWhileRunning         = errors.New("delete while instance running")
	ErrPatchWhileRunning          = errors.New("patch bet settings while instance running")
	ErrPatchSimBetWhileRunning    = errors.New("patch simBet while instance running")
	ErrPatchCurrencyWhileRunning  = errors.New("patch schemeCurrency while instance running")
	ErrInvalidUpdatePatch      = errors.New("invalid update patch")
	ErrFavoriteRequired   = errors.New("favorite required for builtin plan")
)

// builtinPlanLottery 物化时同步到定义行的彩种信息。
type builtinPlanLottery struct {
	Code  string
	Label string
}

// materializeBuiltinPlan 内置计画物化（v8 §3.6 / S1=B 配置复制）：
// 校验快照已收藏 → 复制快照配置（彩种/玩法/方案内容/倍投）→ 记录来源与实际运行类型。
func (s *Service) materializeBuiltinPlan(
	ctx context.Context,
	memberID int64,
	existingConfig []byte,
	snapshotID string,
) (map[string]interface{}, *builtinPlanLottery, error) {
	if snapshotID == "" {
		return nil, nil, fmt.Errorf("%w: snapshotId 不能为空", ErrInvalidUpdatePatch)
	}
	cfg := map[string]interface{}{}
	_ = json.Unmarshal(existingConfig, &cfg)
	if rt, _ := cfg["runTypeId"].(string); NormalizeRunTypeID(rt) != RunTypeBuiltinPlan {
		return nil, nil, fmt.Errorf("%w: 仅内置计划方案可选择收藏方案", ErrInvalidUpdatePatch)
	}

	fav, err := s.q.ExistsMemberSchemeFavorite(ctx, sqlcdb.ExistsMemberSchemeFavoriteParams{
		MemberID:   memberID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, nil, err
	}
	if !fav {
		return nil, nil, ErrFavoriteRequired
	}

	snap, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrSnapshotNotFound
		}
		return nil, nil, err
	}

	snapCfg := map[string]interface{}{}
	_ = json.Unmarshal(snap.Config, &snapCfg)

	overlay := map[string]interface{}{}
	// 复制快照的玩法与方案内容（保留本方案的资金/时间/止盈止损等运行参数）
	for _, key := range []string{
		"playTemplate", "playTypeId", "subPlayId", "typeId", "subId", "betMode", "betUnit", "playMethod",
		"schemeGroups", "jushuList", "triggerBet", "hotColdWarm", "randomDraw", "fixedPick", "rounds", "betMultiplier",
	} {
		if v, ok := snapCfg[key]; ok {
			overlay[key] = v
		}
	}
	overlay["lotteryCode"] = snap.LotteryCode
	actualRunType := RunTypeAdvFixedRotate
	if rt, ok := snapCfg["runTypeId"].(string); ok {
		if normalized := NormalizeRunTypeID(rt); normalized != RunTypeBuiltinPlan {
			actualRunType = normalized
		}
	}
	overlay["builtinPlan"] = map[string]interface{}{
		"snapshotId": snap.ID,
		"runTypeId":  actualRunType,
		"schemeName": snap.SchemeName,
		"playMethod": snap.PlayMethod,
	}
	return overlay, &builtinPlanLottery{Code: snap.LotteryCode, Label: snap.LotteryLabel}, nil
}

var forbiddenUpdateKeys = map[string]struct{}{
	"schemeName":  {},
	"lotteryCode": {},
	"runTypeId":   {},
	"playTypeId":  {},
	"subPlayId":   {},
	"shareStatus": {},
}

type UpdateDefinitionPatch struct {
	RunMode        string
	SimBet         bool
	HasSimBet      bool
	SchemeFunds      string
	SchemeCurrency   string
	HasSchemeCurrency bool
	MultCoeff        string
	HasMultCoeff     bool
	StartTime      string
	EndTime        string
	HasStartTime   bool
	HasEndTime     bool
	SchemeGroups   []string
	StopLoss       string
	TakeProfit     string
	BetUnit        string
	HasBetUnit     bool
	BetMode        string
	HasBetMode     bool
	PlayTemplate   string
	TypeID         string
	SubID          string
	HasCatalogPlay bool
	BetMultiplier  json.RawMessage
	Rounds         json.RawMessage
	HasBetMultiplier bool
	HasRounds        bool
	// 运行类型方案内容（v8 §5）
	JushuList      json.RawMessage
	HasJushuList   bool
	TriggerBet     json.RawMessage
	HasTriggerBet  bool
	HotColdWarm    json.RawMessage
	HasHotColdWarm bool
	RandomDraw     json.RawMessage
	HasRandomDraw  bool
	FixedPick      json.RawMessage
	HasFixedPick   bool
	// 内置计画：选择收藏快照（服务端物化）
	BuiltinPlanSnapshotID string
	HasBuiltinPlan        bool
}

func ParseUpdatePatch(raw map[string]json.RawMessage) (UpdateDefinitionPatch, error) {
	if len(raw) == 0 {
		return UpdateDefinitionPatch{}, fmt.Errorf("%w: 请求体不能为空", ErrInvalidUpdatePatch)
	}
	for key := range raw {
		if _, forbidden := forbiddenUpdateKeys[key]; forbidden {
			return UpdateDefinitionPatch{}, fmt.Errorf("%w: 不可修改字段 %s", ErrInvalidUpdatePatch, key)
		}
	}

	patch := UpdateDefinitionPatch{}
	if v, ok := raw["runMode"]; ok {
		_ = json.Unmarshal(v, &patch.RunMode)
	}
	if v, ok := raw["simBet"]; ok {
		patch.HasSimBet = true
		_ = json.Unmarshal(v, &patch.SimBet)
	}
	if v, ok := raw["schemeFunds"]; ok {
		patch.SchemeFunds = strings.TrimSpace(unquoteJSONString(v))
	}
	if v, ok := raw["schemeCurrency"]; ok {
		patch.HasSchemeCurrency = true
		patch.SchemeCurrency = normalizeSchemeCurrency(unquoteJSONString(v))
	}
	if v, ok := raw["multCoeff"]; ok {
		mc := strings.TrimSpace(unquoteJSONString(v))
		if mc != "" {
			n, err := strconv.ParseInt(mc, 10, 64)
			if err != nil || n < 0 {
				return UpdateDefinitionPatch{}, fmt.Errorf("%w: multCoeff 须为非负整数", ErrInvalidUpdatePatch)
			}
			mc = strconv.FormatInt(n, 10)
		}
		patch.MultCoeff = mc
		patch.HasMultCoeff = true
	}
	if v, ok := raw["startTime"]; ok {
		patch.HasStartTime = true
		_ = json.Unmarshal(v, &patch.StartTime)
	}
	if v, ok := raw["endTime"]; ok {
		patch.HasEndTime = true
		_ = json.Unmarshal(v, &patch.EndTime)
	}
	if v, ok := raw["schemeGroups"]; ok {
		_ = json.Unmarshal(v, &patch.SchemeGroups)
	}
	if v, ok := raw["stopLoss"]; ok {
		patch.StopLoss = strings.TrimSpace(unquoteJSONString(v))
	}
	if v, ok := raw["takeProfit"]; ok {
		patch.TakeProfit = strings.TrimSpace(unquoteJSONString(v))
	}
	if v, ok := raw["betUnit"]; ok {
		patch.BetUnit = strings.TrimSpace(unquoteJSONString(v))
		patch.HasBetUnit = true
	}
	if v, ok := raw["betMode"]; ok {
		patch.BetMode = strings.TrimSpace(unquoteJSONString(v))
		patch.HasBetMode = true
	}
	if v, ok := raw["playTemplate"]; ok {
		patch.PlayTemplate = strings.TrimSpace(unquoteJSONString(v))
		patch.HasCatalogPlay = true
	}
	if v, ok := raw["typeId"]; ok {
		patch.TypeID = strings.TrimSpace(unquoteJSONString(v))
		patch.HasCatalogPlay = true
	}
	if v, ok := raw["subId"]; ok {
		patch.SubID = strings.TrimSpace(unquoteJSONString(v))
		patch.HasCatalogPlay = true
	}
	if v, ok := raw["betMultiplier"]; ok {
		patch.BetMultiplier = append(json.RawMessage(nil), v...)
		patch.HasBetMultiplier = true
	}
	if v, ok := raw["rounds"]; ok {
		patch.Rounds = append(json.RawMessage(nil), v...)
		patch.HasRounds = true
	}
	if v, ok := raw["jushuList"]; ok {
		patch.JushuList = append(json.RawMessage(nil), v...)
		patch.HasJushuList = true
	}
	if v, ok := raw["triggerBet"]; ok {
		patch.TriggerBet = append(json.RawMessage(nil), v...)
		patch.HasTriggerBet = true
	}
	if v, ok := raw["hotColdWarm"]; ok {
		patch.HotColdWarm = append(json.RawMessage(nil), v...)
		patch.HasHotColdWarm = true
	}
	if v, ok := raw["randomDraw"]; ok {
		patch.RandomDraw = append(json.RawMessage(nil), v...)
		patch.HasRandomDraw = true
	}
	if v, ok := raw["fixedPick"]; ok {
		patch.FixedPick = append(json.RawMessage(nil), v...)
		patch.HasFixedPick = true
	}
	if v, ok := raw["builtinPlan"]; ok {
		var bp struct {
			SnapshotID string `json:"snapshotId"`
		}
		if err := json.Unmarshal(v, &bp); err != nil {
			return UpdateDefinitionPatch{}, fmt.Errorf("%w: builtinPlan 格式错误", ErrInvalidUpdatePatch)
		}
		patch.BuiltinPlanSnapshotID = strings.TrimSpace(bp.SnapshotID)
		patch.HasBuiltinPlan = true
	}
	return patch, nil
}

func unquoteJSONString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

func (s *Service) UpdateDefinition(
	ctx context.Context,
	account, definitionID string,
	patch UpdateDefinitionPatch,
) (Definition, error) {
	if s == nil || s.q == nil {
		return Definition{}, ErrUnavailable
	}
	definitionID = strings.TrimSpace(definitionID)
	if definitionID == "" {
		return Definition{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Definition{}, member.ErrNotFound
		}
		return Definition{}, err
	}

	def, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Definition{}, ErrDefinitionNotFound
		}
		return Definition{}, err
	}

	if patch.HasBetMultiplier || patch.HasRounds || patch.HasSchemeCurrency {
		inst, instErr := s.q.GetSchemeInstanceByDefinitionID(ctx, definitionID)
		if instErr == nil && inst.Status == "running" {
			if patch.HasSchemeCurrency && !patch.HasBetMultiplier && !patch.HasRounds {
				return Definition{}, ErrPatchCurrencyWhileRunning
			}
			return Definition{}, ErrPatchWhileRunning
		}
		if instErr != nil && !errors.Is(instErr, pgx.ErrNoRows) {
			return Definition{}, instErr
		}
	}

	// 内置计画：选择收藏快照后服务端物化（配置复制，S1=B）
	var planOverlay map[string]interface{}
	var planLottery *builtinPlanLottery
	if patch.HasBuiltinPlan {
		overlay, lot, perr := s.materializeBuiltinPlan(ctx, m.ID, def.Config, patch.BuiltinPlanSnapshotID)
		if perr != nil {
			return Definition{}, perr
		}
		planOverlay, planLottery = overlay, lot
	}

	cfgBytes, err := mergeUpdateDefinitionConfig(def.Config, patch, planOverlay)
	if err != nil {
		return Definition{}, err
	}

	oldSimBet := configSimBet(def.Config)
	newSimBet := configSimBet(cfgBytes)
	if newSimBet != oldSimBet {
		running, rerr := s.q.HasRunningSchemeInstanceByDefinition(ctx, definitionID)
		if rerr != nil {
			return Definition{}, rerr
		}
		if running {
			return Definition{}, ErrPatchSimBetWhileRunning
		}
	}

	if planLottery != nil {
		if uerr := s.q.UpdateSchemeDefinitionLottery(ctx, sqlcdb.UpdateSchemeDefinitionLotteryParams{
			ID:           definitionID,
			MemberID:     m.ID,
			LotteryCode:  planLottery.Code,
			LotteryLabel: planLottery.Label,
		}); uerr != nil {
			return Definition{}, uerr
		}
	}

	row, err := s.q.UpdateSchemeDefinitionConfig(ctx, sqlcdb.UpdateSchemeDefinitionConfigParams{
		ID:       definitionID,
		MemberID: m.ID,
		Config:   cfgBytes,
	})
	if err != nil {
		return Definition{}, err
	}

	if newSimBet != oldSimBet {
		if _, serr := s.q.SyncSchemeInstancesSimBetByDefinition(ctx, sqlcdb.SyncSchemeInstancesSimBetByDefinitionParams{
			DefinitionID: definitionID,
			SimBet:       newSimBet,
		}); serr != nil {
			return Definition{}, serr
		}
	}

	hasInstance := false
	if _, instErr := s.q.GetSchemeInstanceByDefinitionID(ctx, definitionID); instErr == nil {
		hasInstance = true
	} else if !errors.Is(instErr, pgx.ErrNoRows) {
		return Definition{}, instErr
	}

	return mapDefinitionFromConfigUpdateRow(row, hasInstance), nil
}

func mergeUpdateDefinitionConfig(existing []byte, patch UpdateDefinitionPatch, planOverlay map[string]interface{}) ([]byte, error) {
	base := AddToCloudConfigPatch{
		RunMode:        patch.RunMode,
		SchemeFunds:    patch.SchemeFunds,
		SchemeCurrency: patch.SchemeCurrency,
		StartTime:      patch.StartTime,
		EndTime:        patch.EndTime,
		SchemeGroups:   patch.SchemeGroups,
		StopLoss:       patch.StopLoss,
		TakeProfit:     patch.TakeProfit,
		BetUnit:        patch.BetUnit,
		BetMode:        patch.BetMode,
		PlayTemplate:   patch.PlayTemplate,
		TypeID:         patch.TypeID,
		SubID:          patch.SubID,
	}
	cfgBytes, err := mergeDefinitionConfig(existing, base)
	if err != nil {
		return nil, err
	}
	cfg := map[string]interface{}{}
	_ = json.Unmarshal(cfgBytes, &cfg)
	if patch.HasStartTime {
		if strings.TrimSpace(patch.StartTime) != "" {
			cfg["startTime"] = strings.TrimSpace(patch.StartTime)
		} else {
			delete(cfg, "startTime")
		}
	}
	if patch.HasEndTime {
		if strings.TrimSpace(patch.EndTime) != "" {
			cfg["endTime"] = strings.TrimSpace(patch.EndTime)
		} else {
			delete(cfg, "endTime")
		}
	}
	if patch.HasBetUnit {
		if strings.TrimSpace(patch.BetUnit) != "" {
			cfg["betUnit"] = strings.TrimSpace(patch.BetUnit)
		} else {
			delete(cfg, "betUnit")
		}
	}
	if patch.HasMultCoeff {
		if strings.TrimSpace(patch.MultCoeff) != "" {
			cfg["multCoeff"] = strings.TrimSpace(patch.MultCoeff)
		} else {
			delete(cfg, "multCoeff")
		}
	}
	if patch.HasBetMode {
		if isBetUnitArtifact(patch.BetMode) {
			cfg["betUnit"] = patch.BetMode
			delete(cfg, "betMode")
		} else if strings.TrimSpace(patch.BetMode) != "" {
			cfg["betMode"] = strings.TrimSpace(patch.BetMode)
		} else {
			delete(cfg, "betMode")
		}
	}
	if patch.HasBetMultiplier {
		var v interface{}
		if err := json.Unmarshal(patch.BetMultiplier, &v); err != nil {
			return nil, err
		}
		cfg["betMultiplier"] = v
		// S2：倍投设定统一编译为 rounds 供引擎消费（显式传 rounds 时以 rounds 为准）
		if !patch.HasRounds {
			if m, ok := v.(map[string]interface{}); ok {
				if rounds := compileBetMultiplierRounds(m, cfg); len(rounds) > 0 {
					cfg["rounds"] = rounds
				}
			}
		}
	}
	if patch.HasRounds {
		var v interface{}
		if err := json.Unmarshal(patch.Rounds, &v); err != nil {
			return nil, err
		}
		cfg["rounds"] = v
	}
	if patch.HasJushuList {
		var v interface{}
		if err := json.Unmarshal(patch.JushuList, &v); err != nil {
			return nil, err
		}
		cfg["jushuList"] = v
	}
	if patch.HasTriggerBet {
		var v interface{}
		if err := json.Unmarshal(patch.TriggerBet, &v); err != nil {
			return nil, err
		}
		cfg["triggerBet"] = v
	}
	if patch.HasHotColdWarm {
		var v interface{}
		if err := json.Unmarshal(patch.HotColdWarm, &v); err != nil {
			return nil, err
		}
		cfg["hotColdWarm"] = v
	}
	if patch.HasRandomDraw {
		var v interface{}
		if err := json.Unmarshal(patch.RandomDraw, &v); err != nil {
			return nil, err
		}
		cfg["randomDraw"] = v
	}
	if patch.HasFixedPick {
		var v interface{}
		if err := json.Unmarshal(patch.FixedPick, &v); err != nil {
			return nil, err
		}
		cfg["fixedPick"] = v
	}
	for k, v := range planOverlay {
		cfg[k] = v
	}
	if patch.HasSimBet {
		setConfigSimBet(cfg, patch.SimBet)
	}
	normalizeSchemeConfigBetFields(cfg)
	return json.Marshal(cfg)
}

// compileBetMultiplierRounds 倍投设定 → rounds 编译。
// 简单直线表默认挂翻倍（on_lose）：未中进下一级、命中回第 1 轮；末轮环回第 1 轮。
// advanceMode=on_win（中翻倍）：命中进下一级、未中回第 1 轮。
// 高级倍投（kind=3）使用方案轮次页自定义规则（1-based 局号），首次选择模板时注入默认轮次。
// P1：新保存仅 kind=2（简单直线表）或 kind=3（高级）；kind=0/1 为旧数据兼容，优先 simple.multiples。
func compileBetMultiplierRounds(payload map[string]interface{}, existingCfg map[string]interface{}) []schemeRound {
	kind, _ := payload["kind"].(string)
	if kind == "3" {
		return compileAdvancedBetMultiplierRounds(payload, existingCfg)
	}
	var seq []float64
	advanceMode := "on_lose"
	// 简单表优先（小白/一键已写入 simple.multiples）
	if sm, ok := payload["simple"].(map[string]interface{}); ok {
		if ms, ok := sm["multiples"].(string); ok {
			seq = parseMultSequence(ms)
		}
		if am, ok := sm["advanceMode"].(string); ok && strings.TrimSpace(am) == "on_win" {
			advanceMode = "on_win"
		}
	}
	if len(seq) == 0 {
		switch kind {
		case "0":
			seq = multSeqFromProfitTable(payload["newbie"])
		case "1":
			seq = multSeqFromProfitTable(payload["oneclick"])
		case "2", "":
			// already tried simple above
		}
	}
	if len(seq) == 0 {
		return nil
	}
	rounds := make([]schemeRound, len(seq))
	for i, m := range seq {
		next := i + 1
		if next >= len(seq) {
			next = 0
		}
		if advanceMode == "on_win" {
			rounds[i] = schemeRound{Mult: m, AfterHit: next, AfterMiss: 0}
		} else {
			rounds[i] = schemeRound{Mult: m, AfterHit: 0, AfterMiss: next}
		}
	}
	return rounds
}

// defaultAdvancedBetMultiplierRounds 与客户端 AdvancedSchemeRoundsView 默认表单一致（1-based 跳转）。
func defaultAdvancedBetMultiplierRounds() []schemeRound {
	return []schemeRound{
		{Mult: 0, AfterHit: 2, AfterMiss: 1},
		{Mult: 1, AfterHit: 2, AfterMiss: 3},
		{Mult: 3, AfterHit: 2, AfterMiss: 1},
	}
}

func compileAdvancedBetMultiplierRounds(payload map[string]interface{}, existingCfg map[string]interface{}) []schemeRound {
	adv, ok := payload["advanced"].(map[string]interface{})
	if !ok {
		return nil
	}
	selectedID, _ := adv["selectedId"].(string)
	if strings.TrimSpace(selectedID) == "" {
		return nil
	}
	if raw, ok := adv["rounds"]; ok {
		if rounds := parseSchemeRoundsFromRaw(raw); len(rounds) > 0 {
			return rounds
		}
	}
	if existingCfg != nil {
		prevKind := ""
		if bm, ok := existingCfg["betMultiplier"].(map[string]interface{}); ok {
			prevKind, _ = bm["kind"].(string)
		}
		// 已在高级倍投下保存过轮次（含轮次页编辑）→ 保留，避免覆盖自定义方案
		if prevKind == "3" && len(parseSchemeRoundsFromRaw(existingCfg["rounds"])) > 0 {
			return nil
		}
	}
	return defaultAdvancedBetMultiplierRounds()
}

func multSeqFromProfitTable(raw interface{}) []float64 {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	rows, ok := m["profitTable"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(rows))
	for _, item := range rows {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		switch v := row["mult"].(type) {
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
				out = append(out, f)
			}
		case float64:
			if v > 0 {
				out = append(out, v)
			}
		}
	}
	return out
}

func parseMultSequence(raw string) []float64 {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",", "、", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := strconv.ParseFloat(p, 64)
		if err != nil || f <= 0 {
			continue
		}
		if f > 200000 {
			f = 200000
		}
		out = append(out, f)
	}
	return out
}

func mapDefinitionFromConfigUpdateRow(row sqlcdb.UpdateSchemeDefinitionConfigRow, hasInstance bool) Definition {
	return mapDefinitionFields(
		row.ID, row.Kind, row.SchemeName, row.LotteryCode, row.LotteryLabel,
		row.ShareStatus, row.Config, row.CreatedAt, row.UpdatedAt, hasInstance,
	)
}
