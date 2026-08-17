// import-play-rules converts the approved XLSX rule workbook into reviewable
// draft revisions. It never publishes a rule; that remains an administrator
// action after evaluator and sample-case verification.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"github.com/jackc/pgx/v5/pgtype"
)

func main() {
	file := flag.String("file", "", "path to the XLSX rule workbook")
	dryRun := flag.Bool("dry-run", false, "validate and print matched draft count without writing")
	apply := flag.Bool("apply", false, "insert draft revisions; never publishes rules")
	flag.Parse()
	if *file == "" || (*dryRun == *apply) {
		fmt.Fprintln(os.Stderr, "usage: import-play-rules -file <xlsx> -dry-run|-apply")
		os.Exit(2)
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()
	q := sqlcdb.New(pool)
	candidates, err := q.ListPlayRuleImportCandidates(ctx)
	if err != nil {
		log.Fatalf("list play catalogue: %v", err)
	}
	report, err := ImportFilePartial(*file, catalogCandidates(candidates))
	if err != nil {
		log.Fatalf("validate workbook: %v", err)
	}
	if *dryRun {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"drafts": len(report.Drafts), "unresolved": len(report.Unresolved),
			"ambiguous": len(report.Ambiguous), "mode": "dry-run",
		})
		return
	}
	if len(report.Drafts) == 0 {
		log.Fatal("no safely matched rules to import")
	}
	if err := insertDraftRevisions(ctx, pool, report.Drafts); err != nil {
		log.Fatalf("insert drafts: %v", err)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"draftsCreated": len(report.Drafts), "unresolved": len(report.Unresolved),
		"ambiguous": len(report.Ambiguous), "mode": "apply",
	})
}

func catalogCandidates(rows []sqlcdb.ListPlayRuleImportCandidatesRow) []CatalogCandidate {
	items := make([]CatalogCandidate, 0, len(rows))
	for _, row := range rows {
		items = append(items, CatalogCandidate{
			RuleID: row.RuleID, FullName: row.FullName,
			TemplateCode: row.TemplateCode, TypeID: row.TypeID, SubID: row.SubID,
		})
	}
	return items
}

func insertDraftRevisions(ctx context.Context, pool *db.Pool, drafts []DraftRule) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := sqlcdb.New(tx)
	for _, draft := range drafts {
		revision, err := q.NextPlayRuleSpecRevision(ctx, sqlcdb.NextPlayRuleSpecRevisionParams{
			TemplateCode: draft.TemplateCode, TypeID: draft.TypeID, SubID: draft.SubID, LotteryCode: pgtype.Text{},
		})
		if err != nil {
			return fmt.Errorf("next revision for %s/%s/%s: %w", draft.TemplateCode, draft.TypeID, draft.SubID, err)
		}
		sourceMeta, err := json.Marshal(draft.ExcelRule)
		if err != nil {
			return err
		}
		if _, err := q.InsertPlayRuleSpecRevision(ctx, sqlcdb.InsertPlayRuleSpecRevisionParams{
			TemplateCode: draft.TemplateCode, TypeID: draft.TypeID, SubID: draft.SubID, LotteryCode: pgtype.Text{},
			Revision: revision, Status: "draft", EvaluatorKey: "unclassified", EvaluatorVersion: 1,
			EvaluationSpec: []byte(`{"mode":"unclassified","numberMin":0,"numberMax":0}`),
			SampleCases:    []byte(`[]`), SourceMeta: sourceMeta, Actor: "import-play-rules", ChangeReason: "imported from XLSX workbook",
		}); err != nil {
			return fmt.Errorf("insert draft for %s/%s/%s: %w", draft.TemplateCode, draft.TypeID, draft.SubID, err)
		}
	}
	return tx.Commit(ctx)
}
