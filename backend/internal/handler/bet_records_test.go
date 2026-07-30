package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/cloud/betrecords"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/middleware"
)

// internal/handler 整个包此前没有一个测试文件。
// 投注记录这三个接口是云端中心与会员中心的取数入口，
// 参数校验、越权、错误码映射全发生在这一层，服务层测不到。

type handlerEnv struct {
	h        *Handler
	pool     *db.Pool
	account  string
	memberID int64
	schemeID string
	recordNo string
}

func newHandlerEnv(t *testing.T) *handlerEnv {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)

	account := cfg.ClientDemoAccount
	if account == "" {
		account = "vs8888"
	}
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account = $1`, account).Scan(&memberID); err != nil {
		t.Skipf("会员 %s 不存在：%v", account, err)
	}
	var lottery string
	if err := pool.QueryRow(ctx,
		`SELECT code FROM lottery_catalog WHERE sale_status = 'on_sale' ORDER BY code LIMIT 1`).
		Scan(&lottery); err != nil {
		t.Skipf("没有在售彩种：%v", err)
	}

	// record_no 是 varchar(32)，实例 id 只取纳秒低位给后缀留位置
	schemeID := fmt.Sprintf("hd%d", time.Now().UnixNano()%1e12)
	defID := "d" + schemeID
	recordNo := schemeID + "-R00"
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, schemeID)
		_, _ = pool.Exec(bg, `DELETE FROM scheme_instances WHERE id = $1`, schemeID)
		_, _ = pool.Exec(bg, `DELETE FROM scheme_definitions WHERE id = $1`, defID)
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'handler 测试方案', $3, '测试彩种', 'private', '{}'::jsonb)`,
		defID, memberID, lottery); err != nil {
		t.Fatalf("播方案定义：%v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO scheme_instances (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status)
VALUES ($1, $2, $3, 'custom', 'handler 测试方案', $4, '测试彩种', 'paused')`,
		schemeID, defID, memberID, lottery); err != nil {
		t.Fatalf("播实例：%v", err)
	}
	// 用模拟盘：正式盘的分组查询还要求绑定挂机账号并回填第三方注单号
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_bet_records (
    record_no, member_id, scheme_id, scheme_name, period_no, play_type,
    multiplier, round_label, amount, pnl, status, placed_at, bet_content,
    sim_bet, currency, lottery_code, lottery_label, definition_id, bet_units
) VALUES ($1,$2,$3,'handler 测试方案','20260201001','一星定位胆','1','第1局',
          10, 8, 'hit', now(), '3,7', true, 'CNY', $4, '测试彩种', $5, 2)`,
		recordNo, memberID, schemeID, lottery, defID); err != nil {
		t.Fatalf("播注单：%v", err)
	}

	h := &Handler{
		members:    member.NewService(pool, nil),
		betRecords: betrecords.NewService(pool),
	}
	return &handlerEnv{
		h: h, pool: pool, account: account, memberID: memberID,
		schemeID: schemeID, recordNo: recordNo,
	}
}

// call 带上已登录身份发起请求。pathValues 用于填 {schemeId} 这类路径参数。
func (e *handlerEnv) call(
	fn http.HandlerFunc, target string, pathValues map[string]string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req = req.WithContext(middleware.WithClaims(req.Context(),
		auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: e.account}}))
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

// callAnonymous 不带身份发起请求。
func callAnonymous(fn http.HandlerFunc, target string, pathValues map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

// decodeEnvelope 拆开 apix 信封。
func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) (int, string, json.RawMessage) {
	t.Helper()
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("解析响应失败：%v，原文 %s", err, rec.Body.String())
	}
	return env.Code, env.Message, env.Data
}

// decodeData 取出成功响应里的 data 段。
func decodeData(t *testing.T, rec *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	code, msg, data := decodeEnvelope(t, rec)
	if code != apix.CodeOK {
		t.Fatalf("信封码 = %d（%s），期望 0", code, msg)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("解析 data 失败：%v，原文 %s", err, data)
		}
	}
}

// assertValidationError 断言这是一次参数校验失败。
//
// apix.Validation 走的是「HTTP 200 + 信封码 42200」，不是 HTTP 400，
// 所以校验类的断言必须看信封码；只看 rec.Code 会永远是绿的。
func assertValidationError(t *testing.T, rec *httptest.ResponseRecorder, what string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("%s：HTTP 状态 = %d，参数校验失败按约定应为 200", what, rec.Code)
	}
	code, msg, _ := decodeEnvelope(t, rec)
	if code != apix.CodeValidation {
		t.Fatalf("%s：信封码 = %d（%s），期望 %d", what, code, msg, apix.CodeValidation)
	}
}

// TestBetRecordEndpointsRequireAuth 三个接口都必须先认人。
//
// 路径参数为空的校验发生在 withMember 之前，所以未登录时也会先撞到 400；
// 这里连同这个先后顺序一起钉住，避免有人把校验挪到认证前后而不自知。
func TestBetRecordEndpointsRequireAuth(t *testing.T) {
	h := newHandlerEnv(t).h

	if rec := callAnonymous(h.BetRecordGroups, "/client/cloud/bet-records", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("分组接口未登录返回 %d，期望 401", rec.Code)
	}
	if rec := callAnonymous(h.BetRecordDetail, "/client/cloud/bet-records/x",
		map[string]string{"schemeId": "x"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("明细接口未登录返回 %d，期望 401", rec.Code)
	}
	if rec := callAnonymous(h.BetRecordItem, "/client/cloud/bet-records/item/x",
		map[string]string{"recordNo": "x"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("单笔接口未登录返回 %d，期望 401", rec.Code)
	}
}

// TestBetRecordPathParamValidation 路径参数缺失要报校验失败，而不是当成空串往下查。
func TestBetRecordPathParamValidation(t *testing.T) {
	h := newHandlerEnv(t).h

	assertValidationError(t,
		callAnonymous(h.BetRecordDetail, "/client/cloud/bet-records/", nil), "方案 id 为空")
	// 只有空白也算空
	assertValidationError(t,
		callAnonymous(h.BetRecordItem, "/client/cloud/bet-records/item/%20",
			map[string]string{"recordNo": "   "}), "注单编号全是空白")
}

// TestBetRecordGroupsRejectsBadDateRange 日期区间非法要在入口拦下并说明原因。
func TestBetRecordGroupsRejectsBadDateRange(t *testing.T) {
	env := newHandlerEnv(t)
	for _, tc := range []struct{ name, query string }{
		{name: "日期区间颠倒", query: "?dateFrom=2026-02-10&dateTo=2026-02-01"},
		{name: "日期格式非法", query: "?dateFrom=20260201&dateTo=20260210"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertValidationError(t,
				env.call(env.h.BetRecordGroups, "/client/cloud/bet-records"+tc.query, nil), tc.name)
		})
	}
}

// TestBetRecordGroupsAcceptsUnknownMode 记录一个现存缺口：mode 取值不做校验。
//
// GroupsFilter.Validate() 不看 mode，而 loadRowsFiltered 是拿 mode == "sim" 判真假的，
// 所以 ?mode=typo 会被当成正式盘，接口照常回 200，还把 "typo" 原样写进响应的 mode 字段。
// 前端只会传 real/sim，所以眼下没暴露出来；但拼错一个字母就静悄悄查到另一个盘的数据，
// 不是个好性质。这里按现状钉住——真去收紧校验时，这条测试会提醒你一起改。
func TestBetRecordGroupsAcceptsUnknownMode(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordGroups, "/client/cloud/bet-records?mode=nonsense", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("返回 %d，期望 200", rec.Code)
	}
	var got betrecords.GroupsResult
	decodeData(t, rec, &got)
	if string(got.Mode) != "nonsense" {
		t.Fatalf("mode 字段 = %q，当前实现会把非法取值原样回显", got.Mode)
	}
	// 本测试造的注单是模拟盘的；非法 mode 落到正式盘分支，所以不该出现在结果里
	for _, g := range got.Groups.Items {
		if g.SchemeID == env.schemeID {
			t.Fatal("非法 mode 竟然查到了模拟盘数据，说明分盘判断变了")
		}
	}
}

// TestBetRecordGroupsRejectsBadCursor 游标非法要报校验失败，不能悄悄退回第一页。
func TestBetRecordGroupsRejectsBadCursor(t *testing.T) {
	env := newHandlerEnv(t)
	assertValidationError(t, env.call(env.h.BetRecordGroups,
		"/client/cloud/bet-records?mode=sim&cursor=not-a-number", nil), "游标非法")
}

// TestBetRecordGroupsReturnsSeededScheme 正常查询要把刚播下的方案带回来。
func TestBetRecordGroupsReturnsSeededScheme(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordGroups, "/client/cloud/bet-records?mode=sim&days=3&limit=200", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("返回 %d，期望 200；响应 %s", rec.Code, rec.Body.String())
	}
	var got betrecords.GroupsResult
	decodeData(t, rec, &got)
	for _, g := range got.Groups.Items {
		if g.SchemeID == env.schemeID {
			if g.TotalBet != 10 {
				t.Errorf("投注额 = %.2f，期望 10", g.TotalBet)
			}
			return
		}
	}
	t.Fatalf("分组结果里找不到 %s（共 %d 组）", env.schemeID, len(got.Groups.Items))
}

// TestBetRecordDetailNotFound 查不存在的方案要报 404，而不是回一个空列表。
func TestBetRecordDetailNotFound(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordDetail, "/client/cloud/bet-records/no-such-scheme?mode=sim",
		map[string]string{"schemeId": "no-such-scheme"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("返回 %d，期望 404；响应 %s", rec.Code, rec.Body.String())
	}
}

// TestBetRecordDetailReturnsItems 方案明细要带回该方案下的注单。
func TestBetRecordDetailReturnsItems(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordDetail, "/client/cloud/bet-records/"+env.schemeID+"?mode=sim",
		map[string]string{"schemeId": env.schemeID})
	if rec.Code != http.StatusOK {
		t.Fatalf("返回 %d，期望 200；响应 %s", rec.Code, rec.Body.String())
	}
	var got betrecords.DetailResult
	decodeData(t, rec, &got)
	if len(got.Records.Items) != 1 {
		t.Fatalf("明细应有 1 笔，实际 %d", len(got.Records.Items))
	}
	if got.Records.Items[0].RecordNo != env.recordNo {
		t.Errorf("注单编号 = %q，期望 %q", got.Records.Items[0].RecordNo, env.recordNo)
	}
}

// TestBetRecordItemReturnsDetail 单笔详情要带回完整字段，前端投注详情页直接渲染这份数据。
func TestBetRecordItemReturnsDetail(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordItem, "/client/cloud/bet-records/item/"+env.recordNo,
		map[string]string{"recordNo": env.recordNo})
	if rec.Code != http.StatusOK {
		t.Fatalf("返回 %d，期望 200；响应 %s", rec.Code, rec.Body.String())
	}
	var got betrecords.ItemDetail
	decodeData(t, rec, &got)
	if got.RecordNo != env.recordNo {
		t.Errorf("注单编号 = %q", got.RecordNo)
	}
	if got.Period != "20260201001" {
		t.Errorf("期号 = %q", got.Period)
	}
	if got.BetContent != "3,7" {
		t.Errorf("投注内容 = %q", got.BetContent)
	}
	if !got.SimBet {
		t.Error("模拟盘标记丢了")
	}
}

// TestBetRecordItemNotFound 查不存在的注单要报 404。
func TestBetRecordItemNotFound(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.call(env.h.BetRecordItem, "/client/cloud/bet-records/item/no-such-record",
		map[string]string{"recordNo": "no-such-record"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("返回 %d，期望 404；响应 %s", rec.Code, rec.Body.String())
	}
}

func TestQueryIntFallback(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, query string
		fallback    int
		want        int
	}{
		{name: "参数缺失取默认", query: "/x", fallback: 3, want: 3},
		{name: "空串取默认", query: "/x?days=", fallback: 3, want: 3},
		{name: "非数字取默认", query: "/x?days=abc", fallback: 3, want: 3},
		{name: "正常解析", query: "/x?days=7", fallback: 3, want: 7},
		// 负数原样返回，由下游各自收口
		{name: "负数不在这层拦", query: "/x?days=-1", fallback: 3, want: -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.query, nil)
			if got := queryInt(req, "days", tc.fallback); got != tc.want {
				t.Fatalf("queryInt = %d，期望 %d", got, tc.want)
			}
		})
	}
}
