package games

import "testing"

func TestDisplayBetContent_sscDingwei(t *testing.T) {
	got := displayBetContent("13579,,,,")
	if got != "1 3 5 7 9" {
		t.Fatalf("got %q", got)
	}
}

func TestDisplayBetContent_plain(t *testing.T) {
	got := displayBetContent("01,13,49")
	if got != "01 13 49" {
		t.Fatalf("got %q", got)
	}
}

func TestShortIssueNo(t *testing.T) {
	if got := shortIssueNo("20231103032"); got != "032" {
		t.Fatalf("got %q", got)
	}
}
