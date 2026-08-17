package handler

import (
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
)

// AdminListPlayRules exposes only the currently published rule snapshots.
// This rollout deliberately provides no mutation endpoint.
func (h *Handler) AdminListPlayRules(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	rows, err := sqlcdb.New(h.db).ListPublishedPlayRuleSpecsForAdmin(r.Context())
	if err != nil {
		apix.Internal(w)
		return
	}
	apix.OK(w, map[string]any{"items": playrules.MapAdminRows(rows)})
}
