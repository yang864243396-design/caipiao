package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/guajibet"
)

func guajiWireBetUnits(rule playRule, content string) (int, string) {
	label := rule.BetMode
	typeID := rule.PlayTypeID
	subID := rule.SubPlayID
	typeLabel := ""
	group := ""
	fullName := ""
	switch rule.CatalogSubID {
	case "151":
		typeID, subID = "g009", "151"
		label, typeLabel, group = "\u4e94\u661f\u4e8c\u7801\u4e0d\u5b9a\u4f4d", "\u4e94\u661f", "\u4e94\u661f"
	case "152":
		typeID, subID = "g009", "152"
		label, typeLabel, group = "\u4e94\u661f\u4e09\u7801\u4e0d\u5b9a\u4f4d", "\u4e94\u661f", "\u4e94\u661f"
	case "157":
		typeID, subID = "g015", "157"
		label, typeLabel, group = "\u4e94\u661f\u7ec4\u900960", "\u4e94\u661f", "\u4e94\u661f"
	case "158":
		typeID, subID = "g015", "158"
		label, typeLabel, group = "\u4e94\u661f\u7ec4\u900930", "\u4e94\u661f", "\u4e94\u661f"
	case "87":
		typeID, subID = "g011", "87"
		label, typeLabel, group, fullName = "\u4efb\u4e09\u6df7\u5408\u7ec4\u9009", "\u4efb\u4e09", "\u4efb\u4e09", "\u4efb\u4e09\u6df7\u5408\u7ec4\u9009"
	}
	segmentRule := []byte(`{"guajiGroup":"` + group + `","guajiFullName":"` + fullName + `"}`)
	meta := guajibet.ParseRuleMeta(rule.PlayTemplate, typeID, subID, label, typeLabel, segmentRule, rule.CatalogSubID)
	wire := guajibet.FormatBetContentForRule(meta, content)
	return guajibet.CountBetNums(meta, wire), wire
}

// workerWireBetUnits mirrors the valid outbound worker path: the scheme's
// stored mode wins over name inference, then the formatted wire is resolved
// with the same default amount-unit/multiple inputs used by the worker.
func workerWireBetUnits(rule playRule, content string) (int, string) {
	label := rule.BetMode
	typeID, subID := rule.PlayTypeID, rule.SubPlayID
	typeLabel, group, fullName := "", "", ""
	switch rule.CatalogSubID {
	case "157":
		typeID, subID = "g015", "157"
		label, typeLabel, group = "\u4e94\u661f\u7ec4\u900960", "\u4e94\u661f", "\u4e94\u661f"
	case "158":
		typeID, subID = "g015", "158"
		label, typeLabel, group = "\u4e94\u661f\u7ec4\u900930", "\u4e94\u661f", "\u4e94\u661f"
	case "87":
		typeID, subID = "g011", "87"
		label, typeLabel, group, fullName = "\u4efb\u4e09\u6df7\u5408\u7ec4\u9009", "\u4efb\u4e09", "\u4efb\u4e09", "\u4efb\u4e09\u6df7\u5408\u7ec4\u9009"
	}
	segmentRule := []byte(`{"guajiGroup":"` + group + `","guajiFullName":"` + fullName + `"}`)
	meta := guajibet.ParseRuleMeta(rule.PlayTemplate, typeID, subID, label, typeLabel, segmentRule, rule.CatalogSubID)
	meta.ForcedBetMode = rule.BetMode
	wire := guajibet.FormatBetContentForRule(meta, content)
	return guajibet.ResolveBetsNums(meta, wire, 0, 2, 1), wire
}

func TestFixedRotateWireParity_WuxingBudingweiRejectsLessThanFourDigits(t *testing.T) {
	for _, tc := range []struct{ subID, playMethod, content string }{
		{"151", "\u4e94\u661f\u4e8c\u7801\u4e0d\u5b9a\u4f4d", "0,2"},
		{"152", "\u4e94\u661f\u4e09\u7801\u4e0d\u5b9a\u4f4d", "0,2,4"},
	} {
		t.Run(tc.subID, func(t *testing.T) {
			cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g009","subPlayId":"`+tc.subID+`","catalogSubId":"`+tc.subID+`","betMode":"budingwei"}`), 0, 0)
			if _, err := NormalizeBetPayload(BetPayload{
				PlayTemplate: "ssc_std",
				TypeID:       "g009",
				SubID:        tc.subID,
				BetMode:      "budingwei",
				PlayMethod:   tc.playMethod,
				GroupContent: tc.content,
			}); err == nil {
				t.Errorf("NormalizeBetPayload content=%q want validation error", tc.content)
			}
			if guaji, wire := guajiWireBetUnits(cfg.Play, tc.content); guaji != 0 {
				t.Fatalf("Guaji invalid content=%q wire=%q units=%d want 0", tc.content, wire, guaji)
			}
			if got := planPickBetUnits(cfg, tc.content); got != 0 {
				t.Fatalf("invalid content=%q units=%d want 0", tc.content, got)
			}
		})
	}
}

func TestFixedRotateWireParity_WuxingBudingweiRuleIDsDoNotNeedDisplayLabels(t *testing.T) {
	for _, tc := range []struct {
		subID, content, wantMessage string
	}{
		{"151", "0,2", "五星二码不定位：至少选择 4 个号码"},
		{"152", "0,2,4", "五星三码不定位：至少选择 4 个号码"},
	} {
		t.Run(tc.subID, func(t *testing.T) {
			_, err := NormalizeBetPayload(BetPayload{
				PlayTemplate: "ssc_std",
				TypeID:       "g009",
				SubID:        tc.subID,
				BetMode:      "budingwei",
				GroupContent: tc.content,
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("NormalizeBetPayload content=%q error=%v want %q", tc.content, err, tc.wantMessage)
			}
		})
	}
}

func TestFixedRotateWireParity_WuxingZuFlatPools(t *testing.T) {
	for _, tc := range []struct {
		subID, mode string
		want        int
	}{
		{"157", "zu60", 4},
		{"158", "zu30", 6},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g015","subPlayId":"`+tc.subID+`","catalogSubId":"`+tc.subID+`","betMode":"`+tc.mode+`"}`), 0, 0)
			guaji, wire := workerWireBetUnits(cfg.Play, "0,2,4,6,8")
			if guaji != tc.want {
				t.Fatalf("Guaji %s flat pool wire=%q units=%d want %d", tc.mode, wire, guaji, tc.want)
			}
			if got := countPlayWireBetUnits(cfg.Play, "0,2,4,6,8"); got != guaji {
				t.Fatalf("%s flat pool units=%d want Guaji %d", tc.mode, got, guaji)
			}
		})
	}
}

func TestFixedRotateWireParity_UnknownZuModesKeepFallback(t *testing.T) {
	for _, tc := range []struct {
		name         string
		rule         playRule
		wantFallback int
	}{
		{"custom-zu60", playRule{PlayTemplate: "ssc_std", PlayTypeID: "custom", SubPlayID: "custom", CatalogSubID: "custom", BetMode: "zu60", SegmentLen: 5}, 5},
		{"custom-zu30", playRule{PlayTemplate: "ssc_std", PlayTypeID: "custom", SubPlayID: "custom", CatalogSubID: "custom", BetMode: "zu30", SegmentLen: 5}, 5},
		{"zu60-label-only", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g015", SubPlayID: "zu60", CatalogSubID: "五星组选60", BetMode: "zu60", SegmentLen: 5}, 5},
		{"zu30-label-only", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g015", SubPlayID: "zu30", CatalogSubID: "五星组选30", BetMode: "zu30", SegmentLen: 5}, 5},
		{"zu60-id-outside-wuxing", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g011", SubPlayID: "157", CatalogSubID: "157", BetMode: "zu60", SegmentLen: 5}, 10},
		{"zu30-id-outside-wuxing", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g011", SubPlayID: "158", CatalogSubID: "158", BetMode: "zu30", SegmentLen: 5}, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := parsedSchemeConfig{Play: tc.rule}
			const content = "0,2,4,6,8"
			if got := countPlayWireBetUnits(cfg.Play, content); got != 0 {
				t.Fatalf("unknown %s wire units=%d want 0", tc.rule.BetMode, got)
			}
			if got := planPickBetUnits(cfg, content); got != tc.wantFallback {
				t.Fatalf("unknown %s fallback units=%d want %d", tc.rule.BetMode, got, tc.wantFallback)
			}
		})
	}
}

func TestFixedRotateWireParity_WuxingZuFlatPoolsRecognizeEitherRuleIDField(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule playRule
		want int
	}{
		{"zu60-sub-play-id", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g015", SubPlayID: "157", BetMode: "zu60", SegmentLen: 5}, 4},
		{"zu60-catalog-sub-id", playRule{PlayTemplate: "ssc_std", PlayTypeID: "wuxing", SubPlayID: "zu60", CatalogSubID: "五星组选60 157", BetMode: "zu60", SegmentLen: 5}, 4},
		{"zu30-sub-play-id", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g015", SubPlayID: "158", BetMode: "zu30", SegmentLen: 5}, 6},
		{"zu30-catalog-sub-id", playRule{PlayTemplate: "ssc_std", PlayTypeID: "wuxing", SubPlayID: "zu30", CatalogSubID: "五星组选30 158", BetMode: "zu30", SegmentLen: 5}, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := countPlayWireBetUnits(tc.rule, "0,2,4,6,8"); got != tc.want {
				t.Fatalf("wire units=%d want %d", got, tc.want)
			}
		})
	}
}

func TestFixedRotateWireParity_Ren3HunhePositionMultiplier(t *testing.T) {
	cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"87","catalogSubId":"87","betMode":"hunhe","segmentLen":3}`), 0, 0)
	for _, tc := range []struct {
		name    string
		content string
		want    int
	}{
		{"four-positions-one-ticket", "\u4e07,\u5343,\u767e,\u4e2a\n345", 4},
		{"five-positions-one-ticket", "\u4e07,\u5343,\u767e,\u5341,\u4e2a\n658", 10},
		{"four-positions-two-tickets", "\u4e07,\u5343,\u767e,\u4e2a\n345,346", 8},
		{"five-positions-two-tickets", "\u4e07,\u5343,\u767e,\u5341,\u4e2a\n658,659", 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			guaji, wire := workerWireBetUnits(cfg.Play, tc.content)
			if guaji != tc.want {
				t.Fatalf("Guaji content=%q wire=%q units=%d want %d", tc.content, wire, guaji, tc.want)
			}
			if got := countPlayWireBetUnits(cfg.Play, tc.content); got != guaji {
				t.Fatalf("content=%q units=%d want Guaji %d", tc.content, got, guaji)
			}
		})
	}
}
