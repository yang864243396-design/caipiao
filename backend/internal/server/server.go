package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"caipiao/backend/internal/audit"
	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/cloud/betrecords"
	"caipiao/backend/internal/cloud/instances"
	"caipiao/backend/internal/cloud/lookback"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/content"
	"caipiao/backend/internal/copyhall"
	"caipiao/backend/internal/dashboard"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/games"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/guaji/drawsync"
	"caipiao/backend/internal/guaji/historysync"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/handler"
	"caipiao/backend/internal/maintenance"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/middleware"
	ordersadmin "caipiao/backend/internal/orders/admin"
	"caipiao/backend/internal/orders/bets"
	"caipiao/backend/internal/orders/chases"
	"caipiao/backend/internal/reports"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/ws"
)

type Server struct {
	cfg          config.Config
	authSvc      *auth.Service
	handler      *handler.Handler
	guaji        *guaji.Client
	db           *db.Pool
	mux          *http.ServeMux
	workerCancel context.CancelFunc
}

func New(cfg config.Config) (*Server, error) {
	workerCtx, workerCancel := context.WithCancel(context.Background())

	var pool *db.Pool
	if cfg.DatabaseURL != "" {
		p, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
		if err != nil {
			if cfg.DBRequired {
				workerCancel()
				return nil, fmt.Errorf("database: %w", err)
			}
			slog.Warn("database unavailable, continuing without db", "err", err)
		} else {
			pool = p
			if version, err := p.ServerVersion(context.Background()); err == nil {
				slog.Info("database connected", "postgres", version)
			}
			if err := games.RunLegacyCatalogPurge(context.Background(), pool); err != nil {
				workerCancel()
				return nil, fmt.Errorf("lottery catalog purge: %w", err)
			}
		}
	} else if cfg.DBRequired {
		workerCancel()
		return nil, fmt.Errorf("database: DATABASE_URL 或 DB_HOST 未配置")
	}

	inst := instances.NewStore()

	lb := lookback.NewService(pool)

	betSvc := betrecords.NewService(pool)
	authSvc := auth.NewService(cfg, pool)
	wsHub := ws.NewHub()
	memberSvc := member.NewService(pool, wsHub)
	chaseSvc := chases.NewService(pool, memberSvc)
	copyHallSvc := copyhall.NewService(pool)
	contentSvc := content.NewService(pool)
	auditSvc := audit.NewService(pool)
	maintSvc := maintenance.NewService(pool)
	dashboardSvc := dashboard.NewService(pool)
	ordersAdminSvc := ordersadmin.NewService(pool)
	reportsSvc := reports.NewService(pool)
	wsSrv := &ws.Server{Hub: wsHub, Auth: authSvc, Origins: cfg.CORSOrigins}
	guajiClient := guaji.NewClient(cfg.Guaji)
	guajiAccounts := accountsvc.NewService(pool, guajiClient, cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	periodSyncer := periodsync.NewSyncer(pool, guajiClient, guajiAccounts)
	schemesSvc := schemes.NewService(pool, periodSyncer)
	schemesSvc.SetMemberAuthChecker(guajiAccounts)
	gamesSvc := games.NewService(pool)
	if guajiAccounts != nil {
		gamesSvc.SetGuajiBetPlacer(guajiAccounts)
	}
	var historyWorker *historysync.Worker
	if pool != nil && guajiClient.Enabled() {
		historyWorker = historysync.NewWorker(pool, guajiClient, wsHub)
	}
	gamesSvc.SetDetailDisplaySync(periodSyncer, historyWorker)
	cmsUploads, err := content.NewUploadStore(cfg.CMSUploadDir)
	if err != nil {
		workerCancel()
		return nil, fmt.Errorf("cms upload store: %w", err)
	}
	h := handler.New(authSvc, maintSvc, betSvc, inst, lb, pool, memberSvc, bets.NewService(pool, memberSvc), chaseSvc, copyHallSvc, schemesSvc, gamesSvc, contentSvc, auditSvc, dashboardSvc, ordersAdminSvc, reportsSvc, wsHub, guajiClient, guajiAccounts, cmsUploads)

	var schemeWorker *schemes.Worker
	if pool != nil && cfg.SchemeWorkerEnabled {
		if w := schemes.NewWorker(pool, cfg.SchemeWorkerTickSec, wsHub, periodSyncer); w != nil {
			schemeWorker = w
			if guajiAccounts != nil {
				w.SetGuajiBetPlacer(guajiAccounts)
			}
			go w.Run(workerCtx)
		}
		if bw := bets.NewSettlementWorker(pool, wsHub); bw != nil {
			go bets.RunSettlementWorker(workerCtx, bw, cfg.SchemeWorkerTickSec)
		}
	}
	h.SetMaintenanceResumeScheduler(schemeWorker)

	// T3：第三方开奖 WS 订阅（GUAJI_ENABLED 时；入库与广播不依赖平台 WS 开关）
	if pool != nil && guajiClient.Enabled() {
		if dw := drawsync.NewWorker(pool, guajiClient, wsHub); dw != nil {
			go dw.Run(workerCtx)
		}
	}

	// periods API：同步第三方封盘倒计时（running 彩种，需挂机 token）
	if pool != nil && guajiAccounts != nil && guajiClient.Enabled() {
		if pw := periodsync.NewWorker(pool, guajiClient, guajiAccounts); pw != nil {
			go pw.Run(workerCtx)
		}
	}

	// 历史开奖 REST：同步第三方 §5 开奖历史至 lottery_draws（匿名 GET，无需 token）
	if historyWorker != nil {
		go historyWorker.Run(workerCtx)
	}

	// T6：第三方授权 Token 健康巡检告警（GUAJI_ENABLED 时）
	if guajiAccounts != nil && guajiClient.Enabled() {
		go guajiAccounts.RunTokenMonitor(workerCtx, 5*time.Minute)
	}

	// T5：第三方派奖同步 worker（扫 real pending 注单 → QuerySettlement → 镜像 ledger）
	if guajiAccounts != nil && guajiClient.Enabled() {
		if psw := guajiAccounts.NewPayoutSyncWorker(wsHub, bets.LocalGuajiDrawFallback(pool)); psw != nil {
			if historyWorker != nil {
				psw.SetAfterSettle(func(ctx context.Context, lotteryCode, _ string) {
					_ = historyWorker.SyncLottery(ctx, lotteryCode)
				})
			}
			go psw.Run(workerCtx, 10*time.Second)
		}
	}

	s := &Server{cfg: cfg, authSvc: authSvc, handler: h, guaji: guajiClient, db: pool, workerCancel: workerCancel}
	s.mux = http.NewServeMux()
	s.registerRoutes(wsSrv)
	return s, nil
}

func (s *Server) registerRoutes(wsSrv *ws.Server) {
	api := http.NewServeMux()
	clientAuth := middleware.RequireRole(s.authSvc, auth.RoleClient)
	adminAuth := middleware.RequireRole(s.authSvc, auth.RoleAdmin)

	api.HandleFunc("GET /health", s.handler.Health)
	api.HandleFunc("GET /public/maintenance", s.handler.PublicMaintenance)
	api.HandleFunc("GET /public/lobby-slots", s.handler.PublicLobbySlots)
	api.HandleFunc("GET /public/banners", s.handler.PublicBanners)
	api.HandleFunc("GET /public/site-brand", s.handler.PublicSiteBrand)
	api.HandleFunc("GET /public/cms-uploads/{filename}", s.handler.PublicCMSUpload)
	api.HandleFunc("GET /public/lotteries", s.handler.PublicLotteries)
	api.HandleFunc("GET /public/lotteries/{code}/status", s.handler.PublicLotteryRouteStatus)
	api.HandleFunc("GET /public/lotteries/{code}/play-tree", s.handler.PublicLotteryPlayTree)
	api.HandleFunc("POST /client/auth/login", s.handler.ClientLogin)
	api.HandleFunc("POST /admin/auth/login", s.handler.AdminLogin)
	api.Handle("GET /admin/auth/session", adminAuth(http.HandlerFunc(s.handler.AdminSession)))

	api.Handle("GET /admin/dashboard/kpi", adminAuth(http.HandlerFunc(s.handler.AdminDashboardKpi)))

	api.Handle("GET /admin/members", adminAuth(http.HandlerFunc(s.handler.AdminListMembers)))
	api.Handle("POST /admin/members", adminAuth(http.HandlerFunc(s.handler.AdminCreateMember)))
	api.Handle("GET /admin/members/{memberId}", adminAuth(http.HandlerFunc(s.handler.AdminGetMember)))
	api.Handle("PUT /admin/members/{memberId}", adminAuth(http.HandlerFunc(s.handler.AdminUpdateMember)))
	api.Handle("GET /admin/members/{memberId}/fund-records", adminAuth(http.HandlerFunc(s.handler.AdminMemberFundRecords)))
	api.Handle("POST /admin/members/{memberId}/ops", adminAuth(http.HandlerFunc(s.handler.AdminMemberOp)))
	api.Handle("POST /admin/members/{memberId}/clear-guaji-auth", adminAuth(http.HandlerFunc(s.handler.AdminClearMemberGuajiAuth)))

	api.Handle("GET /admin/orders/bets", adminAuth(http.HandlerFunc(s.handler.AdminListBetOrders)))
	api.Handle("GET /admin/orders/chases", adminAuth(http.HandlerFunc(s.handler.AdminListChaseOrders)))
	api.Handle("GET /admin/orders/ledger", adminAuth(http.HandlerFunc(s.handler.AdminListLedgerEntries)))

	api.Handle("GET /client/cloud/bet-records", clientAuth(http.HandlerFunc(s.handler.BetRecordGroups)))
	api.Handle("GET /client/cloud/bet-records/{schemeId}", clientAuth(http.HandlerFunc(s.handler.BetRecordDetail)))

	api.Handle("GET /client/cloud/schemes/running", clientAuth(http.HandlerFunc(s.handler.CloudRunningSchemes)))
	api.Handle("GET /client/cloud/schemes/stats", clientAuth(http.HandlerFunc(s.handler.CloudCenterStats)))
	api.Handle("GET /client/cloud/lookback", clientAuth(http.HandlerFunc(s.handler.CloudLookbackGet)))
	api.Handle("PUT /client/cloud/lookback", clientAuth(http.HandlerFunc(s.handler.CloudLookbackPut)))
	api.Handle("POST /client/cloud/instances/{instanceId}/start", clientAuth(http.HandlerFunc(s.handler.CloudInstanceStart)))
	api.Handle("POST /client/cloud/instances/{instanceId}/stop", clientAuth(http.HandlerFunc(s.handler.CloudInstanceStop)))
	api.Handle("POST /client/cloud/instances/{instanceId}/pause", clientAuth(http.HandlerFunc(s.handler.CloudInstancePause)))
	api.Handle("POST /client/cloud/instances/{instanceId}/resume", clientAuth(http.HandlerFunc(s.handler.CloudInstanceResume)))
	api.Handle("PUT /client/cloud/instances/{instanceId}/multiplier", clientAuth(http.HandlerFunc(s.handler.CloudInstanceMultiplierPut)))
	api.Handle("PUT /client/cloud/instances/{instanceId}/sim-bet", clientAuth(http.HandlerFunc(s.handler.CloudInstanceSimBetPut)))

	api.Handle("GET /client/member/profile", clientAuth(http.HandlerFunc(s.handler.MemberProfile)))
	api.Handle("GET /client/member/wallet", clientAuth(http.HandlerFunc(s.handler.MemberWallet)))

	api.Handle("GET /client/guaji/auth-status", clientAuth(http.HandlerFunc(s.handler.GuajiAuthStatus)))
	api.Handle("GET /client/guaji/accounts", clientAuth(http.HandlerFunc(s.handler.GuajiListAccounts)))
	api.Handle("POST /client/guaji/accounts/bind", clientAuth(http.HandlerFunc(s.handler.GuajiBindAccount)))
	api.Handle("POST /client/guaji/accounts/{id}/activate", clientAuth(http.HandlerFunc(s.handler.GuajiActivateAccount)))
	api.Handle("POST /client/guaji/accounts/{id}/reauth", clientAuth(http.HandlerFunc(s.handler.GuajiReauthAccount)))
	api.Handle("POST /client/guaji/accounts/{id}/import-session", clientAuth(http.HandlerFunc(s.handler.GuajiImportSession)))
	api.Handle("DELETE /client/guaji/accounts/{id}", clientAuth(http.HandlerFunc(s.handler.GuajiDeleteAccount)))
	api.Handle("GET /client/guaji/balance", clientAuth(http.HandlerFunc(s.handler.GuajiBalance)))
	api.Handle("GET /client/guaji/primary-currency", clientAuth(http.HandlerFunc(s.handler.GuajiGetPrimaryCurrency)))
	api.Handle("PUT /client/guaji/primary-currency", clientAuth(http.HandlerFunc(s.handler.GuajiSetPrimaryCurrency)))

	api.Handle("GET /admin/members/{memberId}/guaji-accounts", adminAuth(http.HandlerFunc(s.handler.AdminMemberGuajiAccounts)))
	api.Handle("GET /admin/guaji/health", adminAuth(http.HandlerFunc(s.handler.AdminGuajiHealth)))
	api.Handle("GET /client/funds/records", clientAuth(http.HandlerFunc(s.handler.FundRecords)))
	api.Handle("GET /client/orders/ledger", clientAuth(http.HandlerFunc(s.handler.OrdersLedger)))
	api.Handle("GET /client/orders/bets", clientAuth(http.HandlerFunc(s.handler.OrdersBets)))
	api.Handle("GET /client/orders/chases", clientAuth(http.HandlerFunc(s.handler.OrdersChases)))

	api.Handle("GET /client/copy-hall/rankings", clientAuth(http.HandlerFunc(s.handler.CopyHallRankings)))

	api.Handle("GET /client/schemes", clientAuth(http.HandlerFunc(s.handler.ListSchemes)))
	api.Handle("GET /client/schemes/check-name", clientAuth(http.HandlerFunc(s.handler.CheckSchemeName)))
	api.Handle("POST /client/schemes", clientAuth(http.HandlerFunc(s.handler.CreateScheme)))
	api.Handle("GET /client/schemes/{definitionId}", clientAuth(http.HandlerFunc(s.handler.GetScheme)))
	api.Handle("PATCH /client/schemes/{definitionId}", clientAuth(http.HandlerFunc(s.handler.PatchScheme)))
	api.Handle("DELETE /client/schemes/{definitionId}", clientAuth(http.HandlerFunc(s.handler.DeleteScheme)))
	api.Handle("PUT /client/schemes/{definitionId}/bet-multiplier", clientAuth(http.HandlerFunc(s.handler.PutBetMultiplier)))
	api.Handle("PUT /client/schemes/{definitionId}/rounds", clientAuth(http.HandlerFunc(s.handler.PutRounds)))
	api.Handle("GET /client/schemes/{definitionId}/bet-multiplier-templates", clientAuth(http.HandlerFunc(s.handler.ClientSchemeTemplates)))
	api.Handle("GET /client/schemes/{definitionId}/bet-multiplier-templates/{templateId}", clientAuth(http.HandlerFunc(s.handler.ClientGetSchemeTemplate)))
	api.Handle("POST /client/schemes/{definitionId}/bet-multiplier-templates", clientAuth(http.HandlerFunc(s.handler.ClientCreateSchemeTemplate)))
	api.Handle("PUT /client/schemes/{definitionId}/bet-multiplier-templates/{templateId}", clientAuth(http.HandlerFunc(s.handler.ClientUpdateSchemeTemplate)))
	api.Handle("POST /client/schemes/{definitionId}/add-to-cloud", clientAuth(http.HandlerFunc(s.handler.AddDefinitionToCloud)))
	api.Handle("POST /client/schemes/{definitionId}/fork-and-add-to-cloud", clientAuth(http.HandlerFunc(s.handler.ForkDefinitionToCloud)))
	api.Handle("GET /client/schemes/favorites", clientAuth(http.HandlerFunc(s.handler.SchemeFavoritesList)))
	api.Handle("POST /client/schemes/favorites", clientAuth(http.HandlerFunc(s.handler.SchemeFavoriteAdd)))
	api.Handle("DELETE /client/schemes/favorites/{snapshotId}", clientAuth(http.HandlerFunc(s.handler.SchemeFavoriteDelete)))
	api.Handle("GET /client/schemes/share-catalog", clientAuth(http.HandlerFunc(s.handler.ShareCatalog)))
	api.Handle("POST /client/schemes/share/{snapshotId}/add-to-cloud", clientAuth(http.HandlerFunc(s.handler.ShareAddToCloud)))
	api.Handle("POST /client/schemes/share/{snapshotId}/follow-bet", clientAuth(http.HandlerFunc(s.handler.ShareFollowBet)))
	api.Handle("POST /client/schemes/contrary/bet", clientAuth(http.HandlerFunc(s.handler.ContraryBet)))
	api.Handle("GET /client/games/lottery-options", clientAuth(http.HandlerFunc(s.handler.MemberLotteryFilterOptions)))
	api.Handle("GET /client/games/{code}/scheme-options", clientAuth(http.HandlerFunc(s.handler.LotterySchemeOptions)))
	api.Handle("GET /client/games/{code}/detail", clientAuth(http.HandlerFunc(s.handler.GameDetail)))
	api.Handle("GET /client/games/{code}/draws", clientAuth(http.HandlerFunc(s.handler.GameDraws)))
	api.Handle("POST /client/games/{code}/bets", clientAuth(http.HandlerFunc(s.handler.GamePlaceBet)))

	api.Handle("GET /client/cloud/global-settings", clientAuth(http.HandlerFunc(s.handler.CloudGlobalSettingsGet)))
	api.Handle("PUT /client/cloud/global-settings", clientAuth(http.HandlerFunc(s.handler.CloudGlobalSettingsPut)))

	api.Handle("GET /client/content/announcements", clientAuth(http.HandlerFunc(s.handler.ContentAnnouncements)))
	api.Handle("GET /client/content/announcements/{id}", clientAuth(http.HandlerFunc(s.handler.ContentAnnouncementDetail)))
	api.Handle("GET /client/content/faq", clientAuth(http.HandlerFunc(s.handler.ContentFaqList)))
	api.Handle("GET /client/content/faq/{id}", clientAuth(http.HandlerFunc(s.handler.ContentFaqDetail)))
	api.Handle("GET /client/content/help", clientAuth(http.HandlerFunc(s.handler.ContentHelpList)))
	api.Handle("POST /client/content/feedback", clientAuth(http.HandlerFunc(s.handler.ContentFeedbackSubmit)))
	api.Handle("GET /client/customer-service/agents", clientAuth(http.HandlerFunc(s.handler.ClientCustomerServiceAgents)))

	api.Handle("GET /admin/content/bundle", adminAuth(http.HandlerFunc(s.handler.AdminContentBundle)))
	api.Handle("GET /admin/content/announcements", adminAuth(http.HandlerFunc(s.handler.AdminListAnnouncements)))
	api.Handle("PUT /admin/content/announcements", adminAuth(http.HandlerFunc(s.handler.AdminSaveAnnouncement)))
	api.Handle("PATCH /admin/content/announcements/{id}/pinned", adminAuth(http.HandlerFunc(s.handler.AdminSetAnnouncementPinned)))
	api.Handle("DELETE /admin/content/announcements/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteAnnouncement)))
	api.Handle("GET /admin/content/faq/articles", adminAuth(http.HandlerFunc(s.handler.AdminListFaqArticles)))
	api.Handle("PUT /admin/content/faq/articles", adminAuth(http.HandlerFunc(s.handler.AdminSaveFaqArticle)))
	api.Handle("DELETE /admin/content/faq/articles/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteFaqArticle)))
	api.Handle("GET /admin/content/help/articles", adminAuth(http.HandlerFunc(s.handler.AdminListHelpArticles)))
	api.Handle("PUT /admin/content/help/articles", adminAuth(http.HandlerFunc(s.handler.AdminSaveHelpArticle)))
	api.Handle("DELETE /admin/content/help/articles/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteHelpArticle)))
	api.Handle("PUT /admin/content/lobby-slots", adminAuth(http.HandlerFunc(s.handler.AdminSaveLobbySlot)))
	api.Handle("GET /admin/content/banners", adminAuth(http.HandlerFunc(s.handler.AdminListBanners)))
	api.Handle("PUT /admin/content/banners", adminAuth(http.HandlerFunc(s.handler.AdminSaveBanner)))
	api.Handle("PATCH /admin/content/banners/{id}/enabled", adminAuth(http.HandlerFunc(s.handler.AdminSetBannerEnabled)))
	api.Handle("DELETE /admin/content/banners/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteBanner)))
	api.Handle("POST /admin/content/uploads/image", adminAuth(http.HandlerFunc(s.handler.AdminUploadCMSImage)))

	api.Handle("GET /admin/service/customer-service/agents", adminAuth(http.HandlerFunc(s.handler.AdminListCustomerServiceAgents)))
	api.Handle("PUT /admin/service/customer-service/agents", adminAuth(http.HandlerFunc(s.handler.AdminSaveCustomerServiceAgent)))
	api.Handle("DELETE /admin/service/customer-service/agents/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteCustomerServiceAgent)))

	api.Handle("GET /admin/operations/maintenance", adminAuth(http.HandlerFunc(s.handler.AdminGetMaintenance)))
	api.Handle("PUT /admin/operations/maintenance", adminAuth(http.HandlerFunc(s.handler.AdminSaveMaintenance)))

	api.Handle("GET /admin/reports/lottery-stat", adminAuth(http.HandlerFunc(s.handler.AdminLotteryStatReport)))
	api.Handle("GET /admin/reports/pnl", adminAuth(http.HandlerFunc(s.handler.AdminPnlReport)))
	api.Handle("GET /admin/reports/daily-lottery", adminAuth(http.HandlerFunc(s.handler.AdminDailyLotteryReport)))

	api.Handle("GET /admin/schemes/instances", adminAuth(http.HandlerFunc(s.handler.AdminSchemeMonitorList)))
	api.Handle("POST /admin/schemes/share", adminAuth(http.HandlerFunc(s.handler.AdminCreateShareSnapshot)))
	api.Handle("PATCH /admin/schemes/share/{snapshotId}", adminAuth(http.HandlerFunc(s.handler.AdminPatchShareSnapshot)))
	api.Handle("DELETE /admin/schemes/share/{snapshotId}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteShareSnapshot)))
	api.Handle("POST /admin/schemes/instances/{instanceId}/force-stop", adminAuth(http.HandlerFunc(s.handler.AdminSchemeForceStop)))
	api.Handle("POST /admin/schemes/instances/{instanceId}/release-stop", adminAuth(http.HandlerFunc(s.handler.AdminSchemeReleaseStop)))

	api.Handle("GET /admin/system/audit-logs", adminAuth(http.HandlerFunc(s.handler.AdminAuditLogs)))
	api.Handle("GET /admin/system/roles", adminAuth(http.HandlerFunc(s.handler.AdminListRoles)))
	api.Handle("PUT /admin/system/roles", adminAuth(http.HandlerFunc(s.handler.AdminSaveRole)))
	api.Handle("DELETE /admin/system/roles/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteRole)))

	api.Handle("GET /admin/system/users", adminAuth(http.HandlerFunc(s.handler.AdminListUsers)))
	api.Handle("POST /admin/system/users", adminAuth(http.HandlerFunc(s.handler.AdminCreateUser)))
	api.Handle("PUT /admin/system/users/{id}", adminAuth(http.HandlerFunc(s.handler.AdminUpdateUser)))
	api.Handle("DELETE /admin/system/users/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteUser)))

	api.Handle("GET /admin/copy-hall/rankings", adminAuth(http.HandlerFunc(s.handler.AdminCopyHallRankings)))
	api.Handle("PUT /admin/copy-hall/boards/{boardKind}", adminAuth(http.HandlerFunc(s.handler.AdminCopyHallSaveBoard)))
	api.Handle("POST /admin/copy-hall/boards/{boardKind}/reset", adminAuth(http.HandlerFunc(s.handler.AdminCopyHallResetBoard)))
	api.Handle("POST /admin/copy-hall/reset-all", adminAuth(http.HandlerFunc(s.handler.AdminCopyHallResetAll)))

	api.Handle("GET /admin/games/lottery-catalog", adminAuth(http.HandlerFunc(s.handler.AdminLotteryCatalogList)))
	api.Handle("GET /admin/games/lottery-catalog/{code}/draws", adminAuth(http.HandlerFunc(s.handler.GameDraws)))
	api.Handle("PATCH /admin/games/lottery-catalog/{code}", adminAuth(http.HandlerFunc(s.handler.AdminLotteryCatalogPatch)))
	api.Handle("GET /admin/games/play-templates", adminAuth(http.HandlerFunc(s.handler.AdminPlayTemplatesList)))
	api.Handle("GET /admin/games/play-templates/{templateCode}/play-tree", adminAuth(http.HandlerFunc(s.handler.AdminPlayTree)))

	api.Handle("GET /admin/games/scheme-templates", adminAuth(http.HandlerFunc(s.handler.AdminListSchemeTemplates)))
	api.Handle("GET /admin/games/scheme-templates/{id}", adminAuth(http.HandlerFunc(s.handler.AdminGetSchemeTemplate)))
	api.Handle("PUT /admin/games/scheme-templates", adminAuth(http.HandlerFunc(s.handler.AdminSaveSchemeTemplate)))
	api.Handle("DELETE /admin/games/scheme-templates/{id}", adminAuth(http.HandlerFunc(s.handler.AdminDeleteSchemeTemplate)))
	api.Handle("POST /admin/games/scheme-templates/reset", adminAuth(http.HandlerFunc(s.handler.AdminResetSchemeTemplates)))

	if s.cfg.WSEnabled && wsSrv != nil {
		api.HandleFunc("GET /ws/public", wsSrv.HandlePublic)
		api.HandleFunc("GET /ws/client", wsSrv.HandleClient)
		api.HandleFunc("GET /ws/admin", wsSrv.HandleAdmin)
	}

	s.mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api))
}

func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	h = middleware.Logger(h)
	h = middleware.Recover(h)
	h = middleware.CORS(s.cfg.CORSOrigins)(h)
	return h
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) Close() {
	if s.workerCancel != nil {
		s.workerCancel()
	}
	if s.db != nil {
		s.db.Close()
	}
}
