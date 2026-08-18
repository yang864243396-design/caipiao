package schemes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeevents"
)

var errInjectedMutation = errors.New("injected mutation failure")

type mutationFaultState struct {
	memberID                   int64
	account                    string
	definitionID               string
	instanceID                 string
	definitionName             string
	instanceName               string
	definitionConfig           []byte
	instanceSimBet             bool
	failRenameInstance         bool
	failDefinitionConfig       bool
	failPostCommitInstanceRead bool
}

func newMutationFaultState() *mutationFaultState {
	return &mutationFaultState{
		memberID:         7,
		account:          "member-7",
		definitionID:     "def-a",
		instanceID:       "inst-a",
		definitionName:   "before",
		instanceName:     "before",
		definitionConfig: []byte(`{"multCoeff":"1","simBet":false}`),
	}
}

func (s *mutationFaultState) clone() mutationFaultState {
	cloned := *s
	cloned.definitionConfig = append([]byte(nil), s.definitionConfig...)
	return cloned
}

func (s *mutationFaultState) service(marker *recordingMarker) *Service {
	db := &mutationFaultDB{state: s}
	return &Service{
		q:        sqlcdb.New(db),
		realtime: marker,
		beginTx: func(context.Context) (pgx.Tx, error) {
			return &mutationFaultTx{root: s, pending: s.clone()}, nil
		},
	}
}

type mutationFaultDB struct {
	state *mutationFaultState
}

func (db *mutationFaultDB) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return executeMutationFault(db.state, false, query, args...)
}

func (db *mutationFaultDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected database query")
}

func (db *mutationFaultDB) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	if db.state.failPostCommitInstanceRead && isInstanceByDefinitionQuery(query) {
		return mutationFaultRow(func(...interface{}) error { return errInjectedMutation })
	}
	return queryMutationFault(db.state, false, query, args...)
}

type mutationFaultTx struct {
	root      *mutationFaultState
	pending   mutationFaultState
	committed bool
}

func (tx *mutationFaultTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected nested transaction")
}

func (tx *mutationFaultTx) Commit(context.Context) error {
	if tx.committed {
		return pgx.ErrTxClosed
	}
	*tx.root = tx.pending.clone()
	tx.committed = true
	return nil
}

func (tx *mutationFaultTx) Rollback(context.Context) error {
	if tx.committed {
		return pgx.ErrTxClosed
	}
	return nil
}

func (tx *mutationFaultTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("unexpected CopyFrom")
}

func (tx *mutationFaultTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *mutationFaultTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *mutationFaultTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("unexpected Prepare")
}

func (tx *mutationFaultTx) Exec(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return executeMutationFault(&tx.pending, true, query, args...)
}

func (tx *mutationFaultTx) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected database query")
}

func (tx *mutationFaultTx) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	return queryMutationFault(&tx.pending, true, query, args...)
}

func (tx *mutationFaultTx) Conn() *pgx.Conn { return nil }

type mutationFaultRow func(...interface{}) error

func (row mutationFaultRow) Scan(dest ...interface{}) error { return row(dest...) }

func queryMutationFault(state *mutationFaultState, inTx bool, query string, args ...interface{}) pgx.Row {
	switch {
	case strings.Contains(query, "FROM members m"):
		return mutationFaultRow(func(dest ...interface{}) error {
			*(dest[0].(*int64)) = state.memberID
			*(dest[1].(*string)) = state.account
			*(dest[2].(*string)) = "Member Seven"
			*(dest[3].(*string)) = "active"
			return nil
		})
	case strings.Contains(query, "EXISTS (SELECT 1 FROM scheme_instances"):
		return mutationFaultRow(func(...interface{}) error { return pgx.ErrNoRows })
	case strings.Contains(query, "FROM scheme_definitions") && strings.Contains(query, "WHERE id = $1 AND member_id = $2"):
		return definitionFaultRow(state)
	case isInstanceByDefinitionQuery(query):
		return instanceByDefinitionFaultRow(state)
	case strings.Contains(query, "UPDATE scheme_definitions") && strings.Contains(query, "config = $3"):
		if state.failDefinitionConfig {
			return mutationFaultRow(func(...interface{}) error { return errInjectedMutation })
		}
		state.definitionConfig = append([]byte(nil), args[2].([]byte)...)
		return updatedDefinitionFaultRow(state)
	case strings.Contains(query, "UPDATE scheme_instances") && strings.Contains(query, "multiplier = $3"):
		return updatedInstanceFaultRow(state, state.instanceSimBet)
	case strings.Contains(query, "UPDATE scheme_instances") && strings.Contains(query, "sim_bet = $3"):
		state.instanceSimBet = args[2].(bool)
		return updatedInstanceFaultRow(state, state.instanceSimBet)
	default:
		return mutationFaultRow(func(...interface{}) error {
			return fmt.Errorf("unexpected query row (inTx=%v): %s", inTx, query)
		})
	}
}

func executeMutationFault(state *mutationFaultState, inTx bool, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if strings.Contains(query, "UPDATE scheme_definitions") && strings.Contains(query, "scheme_name = $3") {
		if strings.Contains(query, "UPDATE scheme_instances") {
			if state.failRenameInstance {
				return pgconn.CommandTag{}, errInjectedMutation
			}
			state.definitionName = args[2].(string)
			state.instanceName = args[2].(string)
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}
		state.definitionName = args[2].(string)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	if strings.Contains(query, "UPDATE scheme_instances") && strings.Contains(query, "scheme_name = $2") {
		if state.failRenameInstance {
			return pgconn.CommandTag{}, errInjectedMutation
		}
		state.instanceName = args[1].(string)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	return pgconn.CommandTag{}, fmt.Errorf("unexpected exec (inTx=%v): %s", inTx, query)
}

func isInstanceByDefinitionQuery(query string) bool {
	return strings.Contains(query, "FROM scheme_instances") && strings.Contains(query, "WHERE definition_id = $1")
}

func definitionFaultRow(state *mutationFaultState) pgx.Row {
	return mutationFaultRow(func(dest ...interface{}) error {
		*(dest[0].(*string)) = state.definitionID
		*(dest[1].(*string)) = "custom"
		*(dest[2].(*string)) = state.definitionName
		*(dest[3].(*string)) = "ssc"
		*(dest[4].(*string)) = "SSC"
		*(dest[5].(*string)) = "private"
		*(dest[8].(*[]byte)) = append([]byte(nil), state.definitionConfig...)
		return nil
	})
}

func updatedDefinitionFaultRow(state *mutationFaultState) pgx.Row {
	return mutationFaultRow(func(dest ...interface{}) error {
		*(dest[0].(*string)) = state.definitionID
		*(dest[1].(*int64)) = state.memberID
		*(dest[2].(*string)) = "custom"
		*(dest[3].(*string)) = state.definitionName
		*(dest[4].(*string)) = "ssc"
		*(dest[5].(*string)) = "SSC"
		*(dest[6].(*string)) = "private"
		*(dest[9].(*[]byte)) = append([]byte(nil), state.definitionConfig...)
		return nil
	})
}

func instanceByDefinitionFaultRow(state *mutationFaultState) pgx.Row {
	return mutationFaultRow(func(dest ...interface{}) error {
		*(dest[0].(*string)) = state.instanceID
		*(dest[1].(*string)) = state.definitionID
		*(dest[2].(*int64)) = state.memberID
		*(dest[3].(*string)) = "custom"
		*(dest[4].(*string)) = state.instanceName
		*(dest[5].(*string)) = "ssc"
		*(dest[6].(*string)) = "SSC"
		*(dest[7].(*string)) = "pending"
		*(dest[13].(*pgtype.Numeric)) = numericFromFloat(1)
		*(dest[15].(*bool)) = state.instanceSimBet
		return nil
	})
}

func updatedInstanceFaultRow(state *mutationFaultState, simBet bool) pgx.Row {
	return mutationFaultRow(func(dest ...interface{}) error {
		*(dest[0].(*string)) = state.instanceID
		*(dest[1].(*string)) = state.definitionID
		*(dest[2].(*int64)) = state.memberID
		*(dest[3].(*string)) = "custom"
		*(dest[4].(*string)) = state.instanceName
		*(dest[5].(*string)) = "ssc"
		*(dest[6].(*string)) = "SSC"
		*(dest[7].(*string)) = "pending"
		*(dest[14].(*pgtype.Numeric)) = numericFromFloat(1)
		*(dest[16].(*bool)) = simBet
		return nil
	})
}

func markerRefs(marker *recordingMarker) []RealtimeInstanceRef {
	marker.mu.Lock()
	defer marker.mu.Unlock()
	return append([]RealtimeInstanceRef(nil), marker.refs...)
}

func TestUpdateDefinitionRollbackKeepsRenameAtomic(t *testing.T) {
	state := newMutationFaultState()
	state.failRenameInstance = true
	marker := &recordingMarker{}
	svc := state.service(marker)

	_, err := svc.UpdateDefinition(context.Background(), state.account, state.definitionID, UpdateDefinitionPatch{
		HasSchemeName: true,
		SchemeName:    "after",
	})
	if !errors.Is(err, errInjectedMutation) {
		t.Fatalf("UpdateDefinition error=%v want injected mutation failure", err)
	}
	if state.definitionName != "before" || state.instanceName != "before" {
		t.Fatalf("committed names definition=%q instance=%q want both before", state.definitionName, state.instanceName)
	}
	if got := markerRefs(marker); len(got) != 0 {
		t.Fatalf("marker refs=%v want none for rolled-back rename", got)
	}
}

func TestUpdateDefinitionMarksCapturedInstanceWhenPostCommitReadWouldFail(t *testing.T) {
	state := newMutationFaultState()
	state.failPostCommitInstanceRead = true
	marker := &recordingMarker{}
	svc := state.service(marker)

	_, err := svc.UpdateDefinition(context.Background(), state.account, state.definitionID, UpdateDefinitionPatch{
		HasMultCoeff: true,
		MultCoeff:    "2",
	})
	if err != nil {
		t.Fatalf("UpdateDefinition: %v", err)
	}
	want := []RealtimeInstanceRef{{MemberID: state.memberID, InstanceID: state.instanceID}}
	if got := markerRefs(marker); !reflect.DeepEqual(got, want) {
		t.Fatalf("marker refs=%v want=%v", got, want)
	}
}

func TestUpdateInstanceSimBetRollbackKeepsInstanceAndDefinitionConsistent(t *testing.T) {
	state := newMutationFaultState()
	state.failDefinitionConfig = true
	marker := &recordingMarker{}
	svc := state.service(marker)

	_, err := svc.UpdateInstanceSimBet(context.Background(), state.account, state.instanceID, true)
	if !errors.Is(err, errInjectedMutation) {
		t.Fatalf("UpdateInstanceSimBet error=%v want injected mutation failure", err)
	}
	if state.instanceSimBet || configSimBet(state.definitionConfig) {
		t.Fatalf("committed simBet instance=%v definition=%v want both false", state.instanceSimBet, configSimBet(state.definitionConfig))
	}
	if got := markerRefs(marker); len(got) != 0 {
		t.Fatalf("marker refs=%v want none for rolled-back simBet", got)
	}
}

type postCommitReadFaultDB struct{}

func (postCommitReadFaultDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected database write")
}

func (postCommitReadFaultDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errInjectedMutation
}

func (postCommitReadFaultDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return mutationFaultRow(func(...interface{}) error { return errInjectedMutation })
}

func TestWorkerCommittedBetMarksWhenPostCommitReadsFail(t *testing.T) {
	marker := &recordingMarker{}
	w := &Worker{q: sqlcdb.New(postCommitReadFaultDB{}), realtime: marker}
	inst := sqlcdb.SchemeInstance{ID: "inst-a", MemberID: 7, Status: "running"}

	w.afterCommittedBet(context.Background(), inst, nil, false)

	want := []RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if got := markerRefs(marker); !reflect.DeepEqual(got, want) {
		t.Fatalf("marker refs=%v want=%v", got, want)
	}
}

func TestWorkerSimSettlementMarksWhenPostCommitReadFails(t *testing.T) {
	marker := &recordingMarker{}
	w := &Worker{q: sqlcdb.New(postCommitReadFaultDB{}), realtime: marker}
	inst := sqlcdb.SchemeInstance{ID: "inst-a", MemberID: 7, Status: "running", SimBet: true}

	w.afterCommittedSimSettlement(context.Background(), inst, nil)

	want := []RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if got := markerRefs(marker); !reflect.DeepEqual(got, want) {
		t.Fatalf("marker refs=%v want=%v", got, want)
	}
}

func TestWorkerCommittedMutationDeduplicatesNestedAndFinalMarks(t *testing.T) {
	marker := &recordingMarker{}
	w := &Worker{realtime: marker}

	w.withCommittedSchemeMarker(context.Background(), RealtimeInstanceRef{MemberID: 7, InstanceID: "inst-a"},
		func(_ context.Context, operationMarker schemeevents.Marker) {
			operationMarker.MarkScheme(7, "inst-a")
		})

	want := []RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if got := markerRefs(marker); !reflect.DeepEqual(got, want) {
		t.Fatalf("marker refs=%v want=%v", got, want)
	}
}
