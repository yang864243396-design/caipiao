// compile-play-rules turns imported rule drafts into executable declarative
// specifications. It does not verify, publish, or alter betting records.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
	"caipiao/backend/internal/schemes"
)

type catalogKey struct{ template, typ, sub string }

func main() {
	dryRun := flag.Bool("dry-run", false, "report compilation results without writing")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cfg := config.Load()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()
	q := sqlcdb.New(pool)
	catalog, err := q.ListPlayRuleImportCandidates(ctx)
	if err != nil {
		log.Fatalf("list catalogue: %v", err)
	}
	type catalogMeta struct{ betMode, fullName string }
	byKey := make(map[catalogKey]catalogMeta, len(catalog))
	for _, item := range catalog {
		byKey[catalogKey{item.TemplateCode, item.TypeID, item.SubID}] = catalogMeta{betMode: item.BetMode, fullName: item.FullName}
	}
	drafts, err := q.ListDraftPlayRuleSpecRevisions(ctx)
	if err != nil {
		log.Fatalf("list drafts: %v", err)
	}
	compiled, skipped := 0, 0
	for _, draft := range drafts {
		key := catalogKey{draft.TemplateCode, draft.TypeID, draft.SubID}
		meta, exists := byKey[key]
		if !exists {
			skipped++
			continue
		}
		result, err := schemes.CompileCatalogRule(schemes.CatalogRuleCompileInput{
			Locator: playrules.Locator{TemplateCode: draft.TemplateCode, TypeID: draft.TypeID, SubID: draft.SubID},
			BetMode: meta.betMode, PlayMethod: meta.fullName,
		})
		if err != nil {
			skipped++
			continue
		}
		spec, err := json.Marshal(result.Spec)
		if err != nil {
			log.Fatalf("marshal spec %s/%s/%s: %v", draft.TemplateCode, draft.TypeID, draft.SubID, err)
		}
		compiled++
		if *dryRun {
			continue
		}
		if _, err := q.UpdateDraftPlayRuleSpecRevisionCompiled(ctx, sqlcdb.UpdateDraftPlayRuleSpecRevisionCompiledParams{
			ID: draft.ID, EvaluatorKey: result.EvaluatorKey, EvaluatorVersion: 1,
			EvaluationSpec: spec, ChangeReason: "compiled from current catalogue resolver",
		}); err != nil {
			log.Fatalf("update draft %d: %v", draft.ID, err)
		}
	}
	_, _ = fmt.Printf("{\"compiled\":%d,\"skipped\":%d,\"mode\":%q}\n", compiled, skipped, map[bool]string{true: "dry-run", false: "apply"}[*dryRun])
}
