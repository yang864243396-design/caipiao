package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestLookbackResetAuditAction(t *testing.T) {
	inst := sqlcdb.SchemeInstance{MemberID: 1, ID: "inst-1", SimBet: false}
	action := lookbackResetAuditAction("individual", inst, "20231103032", 0)
	if action == "" {
		t.Fatal("expected action")
	}
	action = lookbackResetAuditAction("overall", inst, "20231103032", 3)
	if action == "" {
		t.Fatal("expected overall action")
	}
}
