package catalogsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

type LocalLottery struct {
	Code                string
	DisplayName         string
	OutboundLotteryCode string
}

type MatchResult struct {
	Code           string
	OldName        string
	NewName        string
	OldOutbound    string
	NewOutbound    string
	MatchedByName  string
	RemoteID       int
	Changed        bool
	UnmatchedLocal bool
}

type SyncReport struct {
	Matched   []MatchResult
	Unmatched []MatchResult
	RemoteOnly []RemoteLottery
}

// BuildMatchReport 按归一化名称 + code 链/间隔提示将本地彩种与第三方列表对齐。
func BuildMatchReport(local []LocalLottery, remote []RemoteLottery) SyncReport {
	usedRemote := map[int]bool{}
	var matched, unmatched []MatchResult

	for _, row := range local {
		key := localMatchKey(row.Code, row.DisplayName)
		remoteRow, ok := findRemoteForLocal(row.Code, key, remote)
		if !ok {
			unmatched = append(unmatched, MatchResult{
				Code:           row.Code,
				OldName:        row.DisplayName,
				OldOutbound:    row.OutboundLotteryCode,
				UnmatchedLocal: true,
			})
			continue
		}
		usedRemote[remoteRow.ID] = true
		newOutbound := strconv.Itoa(remoteRow.ID)
		newName := strings.TrimSpace(remoteRow.Name)
		changed := row.OutboundLotteryCode != newOutbound || strings.TrimSpace(row.DisplayName) != newName
		matched = append(matched, MatchResult{
			Code:          row.Code,
			OldName:       row.DisplayName,
			NewName:       newName,
			OldOutbound:   row.OutboundLotteryCode,
			NewOutbound:   newOutbound,
			MatchedByName: remoteRow.Name,
			RemoteID:      remoteRow.ID,
			Changed:       changed,
		})
	}

	var remoteOnly []RemoteLottery
	for _, r := range remote {
		if !usedRemote[r.ID] {
			remoteOnly = append(remoteOnly, r)
		}
	}
	applyHashTronFfcOutboundCrossSwap(matched)
	return SyncReport{Matched: matched, Unmatched: unmatched, RemoteOnly: remoteOnly}
}

// hashTronFfcPartners 本地展示名与第三方链标识对调的 1/3/5 分彩对（00115）。
var hashTronFfcPartners = map[string]string{
	"hash_ffc_1m": "tron_ffc_1m",
	"hash_ffc_3m": "tron_ffc_3m",
	"hash_ffc_5m": "tron_ffc_5m",
}

var hashTronFfcCrossSwapped = map[string]bool{
	"hash_ffc_1m": true, "hash_ffc_3m": true, "hash_ffc_5m": true,
	"tron_ffc_1m": true, "tron_ffc_3m": true, "tron_ffc_5m": true,
}

func applyHashTronFfcOutboundCrossSwap(matched []MatchResult) {
	byCode := make(map[string]*MatchResult, len(matched))
	for i := range matched {
		byCode[matched[i].Code] = &matched[i]
	}
	for hashCode, tronCode := range hashTronFfcPartners {
		h, okH := byCode[hashCode]
		t, okT := byCode[tronCode]
		if !okH || !okT {
			continue
		}
		h.NewOutbound, t.NewOutbound = t.NewOutbound, h.NewOutbound
		h.RemoteID, t.RemoteID = t.RemoteID, h.RemoteID
		h.MatchedByName, t.MatchedByName = t.MatchedByName, h.MatchedByName
		h.Changed = h.OldOutbound != h.NewOutbound || strings.TrimSpace(h.OldName) != strings.TrimSpace(h.NewName)
		t.Changed = t.OldOutbound != t.NewOutbound || strings.TrimSpace(t.OldName) != strings.TrimSpace(t.NewName)
	}
}

// Apply 将匹配结果写入 lottery_catalog，并同步关联表 lottery_label。
func Apply(ctx context.Context, pool *db.Pool, report SyncReport) (int, error) {
	if pool == nil {
		return 0, fmt.Errorf("db pool is nil")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	q := sqlcdb.New(tx)
	updated := 0
	for _, m := range report.Matched {
		if !m.Changed {
			continue
		}
		tag, err := tx.Exec(ctx, `
			UPDATE lottery_catalog
			SET display_name = $2, outbound_lottery_code = $3, updated_at = now()
			WHERE code = $1`, m.Code, m.NewName, m.NewOutbound)
		if err != nil {
			return updated, fmt.Errorf("sync %s: %w", m.Code, err)
		}
		if tag.RowsAffected() == 0 {
			return updated, fmt.Errorf("sync %s: row not found", m.Code)
		}
		if strings.TrimSpace(m.OldName) != strings.TrimSpace(m.NewName) {
			labelParams := sqlcdb.UpdateSchemeDefinitionsLotteryLabelParams{
				LotteryCode: m.Code, LotteryLabel: m.NewName,
			}
			if err := q.UpdateSchemeDefinitionsLotteryLabel(ctx, labelParams); err != nil {
				return updated, err
			}
			if err := q.UpdateSchemeInstancesLotteryLabel(ctx, sqlcdb.UpdateSchemeInstancesLotteryLabelParams(labelParams)); err != nil {
				return updated, err
			}
			if err := q.UpdateSchemeShareSnapshotsLotteryLabel(ctx, sqlcdb.UpdateSchemeShareSnapshotsLotteryLabelParams(labelParams)); err != nil {
				return updated, err
			}
		}
		updated++
	}
	if err := tx.Commit(ctx); err != nil {
		return updated, err
	}
	return updated, nil
}

// findRemoteForLocal 在第三方列表中查找与本地 code/名称一致的条目；优先 code 链+间隔提示过滤。
func findRemoteForLocal(code, matchKey string, remote []RemoteLottery) (RemoteLottery, bool) {
	if matchKey == "" {
		return RemoteLottery{}, false
	}
	var loose RemoteLottery
	var looseOK bool
	for _, r := range remote {
		if NormalizeLotteryName(r.Name) != matchKey {
			continue
		}
		if remoteMatchesCodeHints(code, r.Name) {
			return r, true
		}
		if !looseOK {
			loose, looseOK = r, true
		}
	}
	return loose, looseOK
}
