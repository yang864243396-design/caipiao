package schemes

import "testing"

func TestIssueAfter_numeric(t *testing.T) {
	if !issueAfter("10113945000215", "10113945000214") {
		t.Fatal("expected 215 after 214")
	}
	if issueAfter("10113945000214", "10113945000215") {
		t.Fatal("214 should not be after 215")
	}
	if !issueAfter("10113945000215", "0") {
		t.Fatal("expected after 0")
	}
	if issueAfter("10113945000215", "10113945000215") {
		t.Fatal("same issue should not be after")
	}
}
