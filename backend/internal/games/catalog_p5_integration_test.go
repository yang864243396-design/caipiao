package games

import (
	"context"
	"errors"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

const (
	p5TestLotteryCode  = "hash_ffc_1m"
	p5TestSchemeLabel  = "哈希1分彩-P5标签测"
	p5TestSchemeDefID  = "p5-test-scheme-label-def"
	p5TestSchemeInstID = "p5-test-scheme-label-inst"
	p5TestSchemeSnapID = "p5-test-scheme-label-snap"
)

func TestP5MaintenanceLifecycle(t *testing.T) {
	svc, q := openGamesService(t)
	ctx := context.Background()

	orig, err := q.GetLotteryCatalogByCode(ctx, p5TestLotteryCode)
	if err != nil {
		t.Fatalf("get catalog: %v", err)
	}
	t.Cleanup(func() {
		row, err := q.GetLotteryCatalogByCode(ctx, p5TestLotteryCode)
		if err != nil {
			return
		}
		if row.SaleStatus == "maintenance" || row.DisplayName != orig.DisplayName {
			_, _ = svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
				DisplayName:         orig.DisplayName,
				OutboundLotteryCode: textVal(orig.OutboundLotteryCode),
				SortOrder:           int(orig.SortOrder),
				SaleStatus:          "on_sale",
			})
		}
	})

	if orig.SaleStatus == "maintenance" {
		if _, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
			DisplayName:         orig.DisplayName,
			OutboundLotteryCode: textVal(orig.OutboundLotteryCode),
			SortOrder:           int(orig.SortOrder),
			SaleStatus:          "on_sale",
		}); err != nil {
			t.Fatalf("pre-restore on_sale: %v", err)
		}
		orig, err = q.GetLotteryCatalogByCode(ctx, p5TestLotteryCode)
		if err != nil {
			t.Fatalf("reload catalog: %v", err)
		}
	}

	// 上架态不可直接改字段
	_, err = svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		DisplayName: "不应成功",
		SaleStatus:  "on_sale",
	})
	if !errors.Is(err, ErrCatalogNotEditable) {
		t.Fatalf("want ErrCatalogNotEditable, got %v", err)
	}

	// 设为维护
	maint, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		EnterMaintenance: true,
		SaleStatus:       "maintenance",
	})
	if err != nil {
		t.Fatalf("enter maintenance: %v", err)
	}
	if maint.SaleStatus != "maintenance" {
		t.Fatalf("saleStatus=%s", maint.SaleStatus)
	}

	// 公共列表隐藏
	public, err := svc.PublicListLotteries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range public {
		if row.Code == p5TestLotteryCode {
			t.Fatal("maintenance lottery should be hidden from public list")
		}
	}

	// play-tree 维护拦截
	_, err = svc.PublicPlayTree(ctx, p5TestLotteryCode)
	if !errors.Is(err, ErrLotteryMaintenance) {
		t.Fatalf("PublicPlayTree want maintenance, got %v", err)
	}

	// detail 维护拦截
	_, err = svc.Detail(ctx, DetailQuery{LotteryCode: p5TestLotteryCode})
	if !errors.Is(err, ErrLotteryMaintenance) {
		t.Fatalf("Detail want maintenance, got %v", err)
	}

	// 投注维护拦截
	cfg := config.Load()
	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}
	_, err = svc.PlaceBet(ctx, account, p5TestLotteryCode, PlaceBetInput{
		Amount:     1,
		Multiplier: 1,
		BetPayload: schemes.BetPayload{
			PlayTemplate: "pc28_std",
			TypeID:       "pc28_20",
			SubID:        "dxds",
			BetMode:      "dxds",
			GroupContent: "大",
		},
	})
	if !errors.Is(err, ErrLotteryMaintenance) {
		t.Fatalf("PlaceBet want maintenance, got %v", err)
	}

	// 路由状态
	st := svc.PublicLotteryRouteStatus(ctx, p5TestLotteryCode)
	if !st.Exists || st.SaleStatus != "maintenance" {
		t.Fatalf("route status=%+v", st)
	}

	// 维护态改 4 字段
	patched, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		DisplayName:         "哈希1分彩-P5测",
		OutboundLotteryCode: "hash_ffc_1m_p5",
		SortOrder:           int(orig.SortOrder),
		SaleStatus:          "maintenance",
	})
	if err != nil {
		t.Fatalf("patch maintenance fields: %v", err)
	}
	if patched.DisplayName != "哈希1分彩-P5测" || patched.OutboundLotteryCode != "hash_ffc_1m_p5" {
		t.Fatalf("patched=%+v", patched)
	}

	// 恢复上架
	restored, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		DisplayName:         orig.DisplayName,
		OutboundLotteryCode: textVal(orig.OutboundLotteryCode),
		SortOrder:           int(orig.SortOrder),
		SaleStatus:          "on_sale",
	})
	if err != nil {
		t.Fatalf("restore on_sale: %v", err)
	}
	if restored.SaleStatus != "on_sale" {
		t.Fatalf("restored status=%s", restored.SaleStatus)
	}

	_, err = svc.PublicPlayTree(ctx, p5TestLotteryCode)
	if err != nil {
		t.Fatalf("play-tree after restore: %v", err)
	}
}

func TestP5SchemeLabelSync(t *testing.T) {
	svc, q := openGamesService(t)
	cfg := config.Load()
	ctx := context.Background()

	orig, err := q.GetLotteryCatalogByCode(ctx, p5TestLotteryCode)
	if err != nil {
		t.Fatalf("get catalog: %v", err)
	}

	if err := p5EnsureCatalogOnSale(ctx, svc, q, orig); err != nil {
		t.Fatalf("ensure on_sale: %v", err)
	}

	member, err := q.GetMemberByAccount(ctx, cfg.ClientDemoAccount)
	if err != nil {
		t.Skipf("demo member %q: %v", cfg.ClientDemoAccount, err)
	}
	oldLabel := orig.DisplayName
	if err := p5SeedSchemeLabelFixtures(ctx, svc, member.ID, oldLabel); err != nil {
		t.Fatalf("seed fixtures: %v", err)
	}
	t.Cleanup(func() {
		p5RestoreCatalog(ctx, svc, q, orig)
		p5CleanupSchemeLabelFixtures(ctx, svc)
	})

	if _, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		EnterMaintenance: true,
		SaleStatus:       "maintenance",
	}); err != nil {
		t.Fatalf("enter maintenance: %v", err)
	}
	if _, err := svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		DisplayName:         p5TestSchemeLabel,
		OutboundLotteryCode: textVal(orig.OutboundLotteryCode),
		SortOrder:           int(orig.SortOrder),
		SaleStatus:          "maintenance",
	}); err != nil {
		t.Fatalf("patch display name: %v", err)
	}

	def, err := q.GetSchemeDefinitionByID(ctx, p5TestSchemeDefID)
	if err != nil {
		t.Fatalf("get definition: %v", err)
	}
	if def.LotteryLabel != p5TestSchemeLabel {
		t.Fatalf("definition label=%q want %q", def.LotteryLabel, p5TestSchemeLabel)
	}

	inst, err := q.GetSchemeInstanceByID(ctx, p5TestSchemeInstID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if inst.LotteryLabel != p5TestSchemeLabel {
		t.Fatalf("instance label=%q want %q", inst.LotteryLabel, p5TestSchemeLabel)
	}

	snap, err := q.GetSchemeShareSnapshotByID(ctx, p5TestSchemeSnapID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snap.LotteryLabel != p5TestSchemeLabel {
		t.Fatalf("snapshot label=%q want %q", snap.LotteryLabel, p5TestSchemeLabel)
	}
}

func TestP5PublicLotteryRouteStatusCases(t *testing.T) {
	svc, _ := openGamesService(t)
	ctx := context.Background()

	legacy := svc.PublicLotteryRouteStatus(ctx, "tencent_ffc")
	if !legacy.Legacy || legacy.Exists {
		t.Fatalf("legacy=%+v", legacy)
	}

	invalid := svc.PublicLotteryRouteStatus(ctx, "not_a_real_lottery_code")
	if invalid.Exists || invalid.Legacy {
		t.Fatalf("invalid=%+v", invalid)
	}

	onSale := svc.PublicLotteryRouteStatus(ctx, "tron_ffc_1m")
	if !onSale.Exists || onSale.SaleStatus != "on_sale" {
		t.Fatalf("on_sale=%+v", onSale)
	}
}

func openGamesService(t *testing.T) (*Service, *sqlcdb.Queries) {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not configured")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	q := sqlcdb.New(pool)
	return NewService(pool), q
}

func p5EnsureCatalogOnSale(ctx context.Context, svc *Service, q *sqlcdb.Queries, orig sqlcdb.GetLotteryCatalogByCodeRow) error {
	row, err := q.GetLotteryCatalogByCode(ctx, p5TestLotteryCode)
	if err != nil {
		return err
	}
	if row.SaleStatus == "on_sale" && row.DisplayName == orig.DisplayName {
		return nil
	}
	_, err = svc.AdminPatchCatalog(ctx, p5TestLotteryCode, PatchCatalogInput{
		DisplayName:         orig.DisplayName,
		OutboundLotteryCode: textVal(orig.OutboundLotteryCode),
		SortOrder:           int(orig.SortOrder),
		SaleStatus:          "on_sale",
	})
	return err
}

func p5RestoreCatalog(ctx context.Context, svc *Service, q *sqlcdb.Queries, orig sqlcdb.GetLotteryCatalogByCodeRow) {
	_ = p5EnsureCatalogOnSale(ctx, svc, q, orig)
}

func p5SeedSchemeLabelFixtures(ctx context.Context, svc *Service, memberID int64, label string) error {
	if svc.pool == nil {
		return errors.New("pool unavailable")
	}
	p5CleanupSchemeLabelFixtures(ctx, svc)
	_, err := svc.pool.Exec(ctx, `
		INSERT INTO scheme_definitions (
			id, member_id, kind, scheme_name, lottery_code, lottery_label,
			share_status, share_status_locked, config
		) VALUES ($1, $2, 'custom', 'P5标签同步测', $3, $4, 'private', true, '{}')
	`, p5TestSchemeDefID, memberID, p5TestLotteryCode, label)
	if err != nil {
		return err
	}
	_, err = svc.pool.Exec(ctx, `
		INSERT INTO scheme_instances (
			id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status, sim_bet
		) VALUES ($1, $2, $3, 'custom', 'P5标签同步测', $4, $5, 'pending', false)
	`, p5TestSchemeInstID, p5TestSchemeDefID, memberID, p5TestLotteryCode, label)
	if err != nil {
		return err
	}
	_, err = svc.pool.Exec(ctx, `
		INSERT INTO scheme_share_snapshots (
			id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config
		) VALUES ($1, 'custom', 'P5标签同步测', $2, $3, '大小单双', 0, '{}')
	`, p5TestSchemeSnapID, p5TestLotteryCode, label)
	return err
}

func p5CleanupSchemeLabelFixtures(ctx context.Context, svc *Service) {
	if svc.pool == nil {
		return
	}
	_, _ = svc.pool.Exec(ctx, `DELETE FROM scheme_instances WHERE id = $1`, p5TestSchemeInstID)
	_, _ = svc.pool.Exec(ctx, `DELETE FROM scheme_definitions WHERE id = $1`, p5TestSchemeDefID)
	_, _ = svc.pool.Exec(ctx, `DELETE FROM scheme_share_snapshots WHERE id = $1`, p5TestSchemeSnapID)
}
