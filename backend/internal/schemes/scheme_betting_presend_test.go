package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestPreSendFailureMustStillBelongToLockedChain(t *testing.T) {
	failed := sqlcdb.PreSendFailureOutbox{ChainID: "old-chain"}
	newExecution := sqlcdb.SchemeBettingExecutionState{Owner: "event", ChainState: "active", ChainID: "new-chain"}
	if preSendFailureBelongsToExecution(failed, newExecution) {
		t.Fatal("old-chain failure matched a newly rearmed execution chain")
	}
	currentExecution := sqlcdb.SchemeBettingExecutionState{Owner: "event", ChainState: "active", ChainID: "old-chain"}
	if !preSendFailureBelongsToExecution(failed, currentExecution) {
		t.Fatal("current-chain failure did not match its locked execution chain")
	}
}
