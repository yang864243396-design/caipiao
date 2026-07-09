package betrecords

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestThirdPartyBetOrderNo(t *testing.T) {
	if got := thirdPartyBetOrderNo(pgtype.Text{String: " 398698 ", Valid: true}); got != "398698" {
		t.Fatalf("got %q", got)
	}
	if got := thirdPartyBetOrderNo(pgtype.Text{String: "CB123", Valid: true}); got != "" {
		t.Fatalf("non-numeric should be empty, got %q", got)
	}
	if got := thirdPartyBetOrderNo(pgtype.Text{}); got != "" {
		t.Fatalf("empty got %q", got)
	}
}

func TestFormatMultiplierDisplay(t *testing.T) {
	cases := map[string]string{
		"2.0":  "2",
		"1.5":  "2",
		"3":    "3",
		"":     "1",
		"abc":  "abc",
	}
	for in, want := range cases {
		if got := formatMultiplierDisplay(in); got != want {
			t.Fatalf("formatMultiplierDisplay(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFormatRoundDisplay(t *testing.T) {
	if got := formatRoundDisplay("2/3"); got != "2" {
		t.Fatalf("got %q", got)
	}
	if got := formatRoundDisplay("1"); got != "1" {
		t.Fatalf("got %q", got)
	}
}
