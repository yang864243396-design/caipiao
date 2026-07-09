package catalogsync

import "testing"

func TestRemoteMatchesCodeHints_tronVsHash(t *testing.T) {
	if !remoteMatchesCodeHints("tron_ffc_1m", "波场一分彩") {
		t.Fatal("tron should match 波场一分彩")
	}
	if remoteMatchesCodeHints("tron_ffc_1m", "哈希一分彩") {
		t.Fatal("tron must not match 哈希一分彩")
	}
	if !remoteMatchesCodeHints("hash_ffc_1m", "哈希一分彩") {
		t.Fatal("hash should match 哈希一分彩")
	}
	if remoteMatchesCodeHints("hash_ffc_1m", "波场一分彩") {
		t.Fatal("hash must not match 波场一分彩")
	}
}

func TestFindRemoteForLocal_prefersChainHint(t *testing.T) {
	remote := []RemoteLottery{
		{ID: 21, Name: "哈希一分彩"},
		{ID: 27, Name: "波场一分彩"},
	}
	key := localMatchKey("tron_ffc_1m", "波场1分彩")
	got, ok := findRemoteForLocal("tron_ffc_1m", key, remote)
	if !ok || got.ID != 27 {
		t.Fatalf("got=%+v ok=%v want id=27", got, ok)
	}
}

func TestIntervalHintFromCode(t *testing.T) {
	if got := intervalHintFromCode("tron_ffc_1m"); got != "1分" {
		t.Fatalf("got %q", got)
	}
	if got := intervalHintFromCode("tron_ffc_3m"); got != "3分" {
		t.Fatalf("got %q", got)
	}
}
