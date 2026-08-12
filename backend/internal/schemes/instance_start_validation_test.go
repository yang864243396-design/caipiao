package schemes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

type startValidationDB struct {
	rows []pgx.Row
	next int
}

func (db *startValidationDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected database write")
}

func (db *startValidationDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected database query")
}

func (db *startValidationDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	if db.next >= len(db.rows) {
		return startValidationRow(func(...interface{}) error {
			return fmt.Errorf("unexpected query row %d", db.next+1)
		})
	}
	row := db.rows[db.next]
	db.next++
	return row
}

type startValidationRow func(...interface{}) error

func (row startValidationRow) Scan(dest ...interface{}) error {
	return row(dest...)
}

func TestStartInstanceRejectsLegacyInvalidFixedRotateGroup(t *testing.T) {
	config := []byte(`{
		"runTypeId":"fixed_rotate",
		"playTemplate":"ssc_std",
		"playTypeId":"g009",
		"subPlayId":"151",
		"catalogSubId":"151",
		"betMode":"budingwei",
		"playMethodLabel":"五星二码不定位",
		"schemeCurrency":"USDT",
		"betUnit":"0.001",
		"schemeGroups":["0,2"],
		"rounds":[{"mult":1,"afterHit":0,"afterMiss":0}]
	}`)
	db := &startValidationDB{rows: []pgx.Row{
		startValidationRow(func(dest ...interface{}) error {
			*(dest[0].(*int64)) = 42
			*(dest[1].(*string)) = "member-1"
			*(dest[2].(*string)) = "Member One"
			*(dest[3].(*string)) = "active"
			return nil
		}),
		startValidationRow(func(dest ...interface{}) error {
			*(dest[0].(*string)) = "instance-1"
			*(dest[1].(*string)) = "definition-1"
			*(dest[2].(*int64)) = 42
			*(dest[3].(*string)) = "custom"
			*(dest[4].(*string)) = "Legacy invalid fixed rotate"
			*(dest[5].(*string)) = "ssc"
			*(dest[6].(*string)) = "SSC"
			*(dest[7].(*string)) = "pending"
			*(dest[15].(*pgtype.Numeric)) = numericFromFloat(1)
			*(dest[17].(*bool)) = true
			return nil
		}),
		startValidationRow(func(dest ...interface{}) error {
			*(dest[0].(*string)) = "definition-1"
			*(dest[1].(*string)) = "custom"
			*(dest[2].(*string)) = "Legacy invalid fixed rotate"
			*(dest[3].(*string)) = "ssc"
			*(dest[4].(*string)) = "SSC"
			*(dest[8].(*[]byte)) = config
			return nil
		}),
	}}
	service := &Service{q: sqlcdb.New(db)}

	_, err := service.StartInstance(context.Background(), "member-1", "instance-1")
	if err == nil {
		t.Fatal("StartInstance error=nil want fixed-rotate content validation error")
	}
	if !strings.Contains(err.Error(), "第 1 组：五星二码不定位：至少选择 4 个号码") {
		t.Fatalf("StartInstance error=%q want fixed-rotate content detail", err)
	}
	if errors.Is(err, ErrMinBetAmountTooLow) {
		t.Fatalf("StartInstance returned misleading minimum-amount error: %v", err)
	}
	if db.next != 3 {
		t.Fatalf("query rows consumed=%d want 3", db.next)
	}
}

func TestValidateFixedRotateConfigForStartIgnoresOtherRunModes(t *testing.T) {
	config := []byte(`{
		"runTypeId":"fixed_number",
		"playTemplate":"ssc_std",
		"playTypeId":"g009",
		"subPlayId":"151",
		"catalogSubId":"151",
		"betMode":"budingwei",
		"schemeGroups":["0,2"]
	}`)
	if err := validateFixedRotateConfigForStart("custom", config); err != nil {
		t.Fatalf("non-fixed-rotate start validation error=%v want nil", err)
	}
}
