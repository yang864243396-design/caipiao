package catalogsync

import (
	"strconv"
	"testing"
)

func TestBuildMatchReport_iyesDev_tronHash1m(t *testing.T) {
	remote := []RemoteLottery{
		{ID: 21, Name: "哈希一分彩"},
		{ID: 27, Name: "波场一分彩"},
	}
	local := []LocalLottery{
		{Code: "hash_ffc_1m", DisplayName: "哈希1分彩", OutboundLotteryCode: "4"},
		{Code: "tron_ffc_1m", DisplayName: "波场1分彩", OutboundLotteryCode: "29"},
	}
	report := BuildMatchReport(local, remote)
	byCode := map[string]MatchResult{}
	for _, m := range report.Matched {
		byCode[m.Code] = m
	}
	if m := byCode["hash_ffc_1m"]; m.NewOutbound != "27" || m.RemoteID != 27 {
		t.Fatalf("hash_ffc_1m: %+v want outbound=27", m)
	}
	if m := byCode["tron_ffc_1m"]; m.NewOutbound != "21" || m.RemoteID != 21 {
		t.Fatalf("tron_ffc_1m: %+v want outbound=21", m)
	}
}

func TestBuildMatchReport_iyesDev_allPlatformLotteries(t *testing.T) {
	remote := IyesDevRemoteLotteries()
	var local []LocalLottery
	for code, wantID := range IyesDevOutboundByCode {
		name := seedDisplayNames[code]
		if name == "" {
			t.Fatalf("missing seedDisplayNames for %s", code)
		}
		local = append(local, LocalLottery{
			Code:                code,
			DisplayName:         name,
			OutboundLotteryCode: strconv.Itoa(DocOutboundByCode[code]), // 模拟 00087 错误序号
		})
		_ = wantID
	}
	report := BuildMatchReport(local, remote)
	if len(report.Unmatched) > 0 {
		for _, u := range report.Unmatched {
			t.Errorf("unmatched local: %s %s", u.Code, u.OldName)
		}
		t.Fatal("all platform lotteries should match iyes.dev remote list")
	}
	byCode := map[string]MatchResult{}
	for _, m := range report.Matched {
		byCode[m.Code] = m
	}
	for code, wantID := range IyesDevOutboundByCode {
		m, ok := byCode[code]
		if !ok {
			t.Fatalf("missing match for %s", code)
		}
		if m.NewOutbound != strconv.Itoa(wantID) || m.RemoteID != wantID {
			t.Fatalf("%s: got outbound=%s remote=%d want %d (%s)",
				code, m.NewOutbound, m.RemoteID, wantID, iyesDevRemoteNames[wantID])
		}
		if hashTronFfcCrossSwapped[code] {
			continue
		}
		if !remoteMatchesCodeHints(code, m.MatchedByName) {
			t.Fatalf("%s: remote name %q failed chain/interval hints", code, m.MatchedByName)
		}
	}
}

func TestIyesDevOutbound_differsFromDocIndex(t *testing.T) {
	var diff int
	for code, iyesID := range IyesDevOutboundByCode {
		docID, ok := DocOutboundByCode[code]
		if !ok {
			continue
		}
		if docID != iyesID {
			diff++
		}
	}
	if diff < 40 {
		t.Fatalf("expected most iyes ids to differ from doc §8, got %d diffs", diff)
	}
}

func TestRemoteMatchesCodeHints_preventsCrossChainSwap(t *testing.T) {
	pairs := []struct {
		code       string
		remoteName string
		wantOK     bool
	}{
		{"hash_ffc_1m", "哈希一分彩", true},
		{"hash_ffc_1m", "波场一分彩", false},
		{"tron_ffc_1m", "波场一分彩", true},
		{"tron_ffc_1m", "哈希一分彩", false},
		{"eth_ffc_1m", "以太坊一分彩", true},
		{"eth_ffc_1m", "波场一分彩", false},
		{"eth_pk10_jisu", "以太极速赛车", true},
		{"eth_pk10_jisu", "波场极速赛车", false},
		{"bnb_k3_3m", "币安3分快三", true},
		{"bnb_k3_3m", "波场3分快三", false},
		{"tron_lhc_1m", "波场1分六合彩", true},
		{"tron_lhc_1m", "波场六合彩", false},
		{"hash_ffc_1m", "哈希一分彩", true},
		{"hash_ffc_1m", "波场一分彩", false},
	}
	for _, p := range pairs {
		got := remoteMatchesCodeHints(p.code, p.remoteName)
		if got != p.wantOK {
			t.Fatalf("%s vs %q: got %v want %v", p.code, p.remoteName, got, p.wantOK)
		}
	}
}
