package games

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// T2：彩种 + 玩法 → 第三方 game_id/rule_id 解析。
func TestResolveOutboundFromPlayTree(t *testing.T) {
	svc, _ := openGamesService(t)
	ctx := context.Background()

	const code = "tron_ffc_1m"
	tree, err := svc.PublicPlayTree(ctx, code)
	if err != nil {
		t.Skipf("play-tree %s 不可用（需先 migrate+purge）: %v", code, err)
	}
	if len(tree.PlayTypes) == 0 || len(tree.PlayTypes[0].SubPlays) == 0 {
		t.Fatalf("play-tree %s 无玩法/子玩法", code)
	}
	typeID := tree.PlayTypes[0].TypeID
	sub := tree.PlayTypes[0].SubPlays[0]

	out, err := svc.ResolveOutbound(ctx, code, typeID, sub.SubID)
	if err != nil {
		t.Fatalf("ResolveOutbound: %v", err)
	}
	if out.GameID == "" {
		t.Fatalf("GameID 为空")
	}
	if out.RuleID == "" {
		t.Fatalf("RuleID 为空")
	}
	// RuleID 应等于子玩法 outbound_play_code，并含子玩法标识。
	if out.RuleID != sub.OutboundPlayCode && !strings.Contains(out.RuleID, sub.SubID) {
		t.Fatalf("RuleID=%q 与子玩法 %q/%q 不一致", out.RuleID, sub.OutboundPlayCode, sub.SubID)
	}
	if out.PlayTemplate != tree.PlayTemplate {
		t.Fatalf("PlayTemplate=%q want %q", out.PlayTemplate, tree.PlayTemplate)
	}
}

func TestResolveOutboundUnknownSubPlay(t *testing.T) {
	svc, _ := openGamesService(t)
	ctx := context.Background()

	const code = "tron_ffc_1m"
	if _, err := svc.PublicPlayTree(ctx, code); err != nil {
		t.Skipf("play-tree %s 不可用: %v", code, err)
	}
	_, err := svc.ResolveOutbound(ctx, code, "no_such_type", "no_such_sub")
	if !errors.Is(err, ErrSubPlayNotFound) {
		t.Fatalf("want ErrSubPlayNotFound, got %v", err)
	}
}

func TestResolveOutboundLegacyOrMissing(t *testing.T) {
	svc, _ := openGamesService(t)
	ctx := context.Background()

	_, err := svc.ResolveOutbound(ctx, "tencent_ffc", "qian3", "x")
	if !errors.Is(err, ErrLotteryNotFound) {
		t.Fatalf("legacy/missing want ErrLotteryNotFound, got %v", err)
	}
}
