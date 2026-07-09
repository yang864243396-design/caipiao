package accountsvc

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestAccountTokenValidExpiredAt(t *testing.T) {
	r := row{
		accessTokenEnc: pgtype.Text{String: "enc", Valid: true},
		tokenExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
	}
	if accountTokenValid(r) {
		t.Fatal("expected expired token")
	}
	acct := mapPublic(row{
		id:             1,
		guajiUsername:  "u1",
		isActive:       true,
		accessTokenEnc: r.accessTokenEnc,
		tokenExpiresAt: r.tokenExpiresAt,
		boundAt:        time.Now(),
	})
	if !acct.AuthExpired {
		t.Fatal("expected authExpired on active row with expired token")
	}
}

func TestAccountTokenValidMissingAccess(t *testing.T) {
	r := row{isActive: true}
	if accountTokenValid(r) {
		t.Fatal("expected invalid without access token")
	}
}
