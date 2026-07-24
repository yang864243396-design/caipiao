package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/guajibet"
)

// TestTriggerBetQian3FushiAltPosPrev64695 回归：上期 64695、前三复式按位开某投某。
// 不得把万位开出 6 的整行反投「6,7\n6,7\n6,7」压成 67,67,67；
// 应按万/千/百各自开出 6/4/6 取对应位：正投 68,46,68 / 反投 67,45,67。
func TestTriggerBetQian3FushiAltPosPrev64695(t *testing.T) {
	t.Parallel()
	rows := make([]string, 0, 10)
	for i := 0; i <= 9; i++ {
		d := string(rune('0' + i))
		posCell := d + "," + string(rune('0'+((i+2)%10)))
		negCell := d + "," + string(rune('0'+((i+1)%10)))
		pos := strings.Join([]string{posCell, posCell, posCell}, `\n`)
		neg := strings.Join([]string{negCell, negCell, negCell}, `\n`)
		rows = append(rows, `{"enabled":true,"open":"`+d+`","pos":"`+pos+`","neg":"`+neg+`"}`)
	}
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g001",
		"subPlayId":"1",
		"betMode":"fushi",
		"triggerBet":{
			"mode":"alt_pos_first",
			"rows":[` + strings.Join(rows, ",") + `]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if !triggerBetUsesPosition(cfg.Play) || cfg.Play.SegmentLen != 3 {
		t.Fatalf("play=%+v usesPos=%v", cfg.Play, triggerBetUsesPosition(cfg.Play))
	}
	balls := []string{"6", "4", "6", "9", "5"}
	meta := guajibet.ParseRuleMeta("ssc_std", "g001", "1", "前三直选复式", "前三码", nil, "1")

	posDec := resolveTriggerBetDecision(cfg, balls, "neg") // 上一局反投 → 本期正投
	if posDec.Direction != "pos" || posDec.Content != "6,8\n4,6\n6,8" {
		t.Fatalf("pos pick dir=%q content=%q", posDec.Direction, posDec.Content)
	}
	if wire := guajibet.FormatBetContentForRule(meta, posDec.Content); wire != "68,46,68" {
		t.Fatalf("pos wire=%q want 68,46,68", wire)
	}

	negDec := resolveTriggerBetDecision(cfg, balls, "pos") // 上一局正投 → 本期反投
	if negDec.Direction != "neg" || negDec.Content != "6,7\n4,5\n6,7" {
		t.Fatalf("neg pick dir=%q content=%q", negDec.Direction, negDec.Content)
	}
	if wire := guajibet.FormatBetContentForRule(meta, negDec.Content); wire != "67,45,67" {
		t.Fatalf("neg wire=%q want 67,45,67 (bug form was 67,67,67)", wire)
	}
}
