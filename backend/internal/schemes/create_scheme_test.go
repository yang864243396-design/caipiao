package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/member"
)

// CreateDefinition 此前零覆盖——全仓库搜不到一处引用。
// 它是方案生命周期的入口，且内部有查彩种、查子玩法、内置计划放宽这几条 DB 分支。

func TestValidateCreateInput(t *testing.T) {
	t.Parallel()
	base := CreateDefinitionInput{
		Kind: "custom", LotteryCode: "tron_ffc_1m",
		RunTypeID: "fixed_number", PlayTypeID: "g006", SubPlayID: "13",
	}
	withField := func(mutate func(*CreateDefinitionInput)) CreateDefinitionInput {
		in := base
		mutate(&in)
		return in
	}

	cases := []struct {
		name    string
		in      CreateDefinitionInput
		wantErr bool
		// wantMsg 非空时要求错误里带上这段说明，避免把 A 的报错错认成 B 的
		wantMsg string
	}{
		{name: "完整输入", in: base},
		{
			name: "kind 非法", wantErr: true, wantMsg: "kind",
			in: withField(func(in *CreateDefinitionInput) { in.Kind = "unknown" }),
		},
		{
			name: "kind 允许 contrary",
			in:   withField(func(in *CreateDefinitionInput) { in.Kind = "contrary" }),
		},
		{
			name: "kind 允许 follow",
			in:   withField(func(in *CreateDefinitionInput) { in.Kind = "follow" }),
		},
		{
			name: "方案名 128 字刚好放行",
			in: withField(func(in *CreateDefinitionInput) {
				in.SchemeName = strings.Repeat("方", 128)
			}),
		},
		{
			name: "方案名 129 字超长", wantErr: true, wantMsg: "schemeName",
			in: withField(func(in *CreateDefinitionInput) {
				in.SchemeName = strings.Repeat("方", 129)
			}),
		},
		{
			name: "runTypeId 为空", wantErr: true, wantMsg: "runTypeId",
			in: withField(func(in *CreateDefinitionInput) { in.RunTypeID = "" }),
		},
		{
			name: "彩种为空", wantErr: true, wantMsg: "lotteryCode",
			in: withField(func(in *CreateDefinitionInput) { in.LotteryCode = "" }),
		},
		{
			name: "玩法为空", wantErr: true, wantMsg: "playTypeId",
			in: withField(func(in *CreateDefinitionInput) { in.PlayTypeID = "" }),
		},
		{
			name: "子玩法为空", wantErr: true, wantMsg: "subPlayId",
			in: withField(func(in *CreateDefinitionInput) { in.SubPlayID = "" }),
		},
		{
			// 内置计划的彩种与玩法要等选了收藏方案才物化出来，创建时必须放行
			name: "内置计划可不带彩种与玩法",
			in: withField(func(in *CreateDefinitionInput) {
				in.RunTypeID = RunTypeBuiltinPlan
				in.LotteryCode, in.PlayTypeID, in.SubPlayID = "", "", ""
			}),
		},
		{
			// 放宽只对 custom 生效
			name: "跟单方案的内置计划仍要求彩种", wantErr: true, wantMsg: "lotteryCode",
			in: withField(func(in *CreateDefinitionInput) {
				in.Kind = "follow"
				in.RunTypeID = RunTypeBuiltinPlan
				in.LotteryCode = ""
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCreateInput(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatal("应当报错，实际通过")
				}
				if !errors.Is(err, ErrInvalidCreateRequest) {
					t.Fatalf("错误类型应为 ErrInvalidCreateRequest，实际 %v", err)
				}
				if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
					t.Fatalf("错误里应提到 %q，实际 %v", tc.wantMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("应当通过，实际 %v", err)
			}
		})
	}
}

func TestNormalizeCreateInputTrimsSpace(t *testing.T) {
	t.Parallel()
	in := CreateDefinitionInput{
		Kind: " custom ", SchemeName: "  我的方案  ", LotteryCode: " tron_ffc_1m ",
		RunTypeID: " fixed_number ", PlayTypeID: " g006 ", SubPlayID: " 13 ",
	}
	normalizeCreateInput(&in)
	want := CreateDefinitionInput{
		Kind: "custom", SchemeName: "我的方案", LotteryCode: "tron_ffc_1m",
		RunTypeID: "fixed_number", PlayTypeID: "g006", SubPlayID: "13",
	}
	if in != want {
		t.Fatalf("去空白后 %+v，期望 %+v", in, want)
	}
}

// createEnv 建方案测试的 DB 环境。
type createEnv struct {
	svc     *Service
	pool    *db.Pool
	account string
	lottery string
	// 该彩种模板下确实存在的一个玩法
	playTypeID, subPlayID string
}

func newCreateEnv(t *testing.T) *createEnv {
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
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM members WHERE account = $1)`, account).Scan(&exists); err != nil || !exists {
		t.Skipf("会员 %s 不存在", account)
	}

	var lotteryCode, typeID, subID string
	err = pool.QueryRow(ctx, `
SELECT c.code, sp.type_id, sp.sub_id
FROM lottery_catalog c
JOIN sub_plays sp ON sp.template_code = c.play_template
WHERE c.sale_status = 'on_sale' AND sp.enabled AND sp.label = '一星定位胆'
ORDER BY c.code
LIMIT 1`).Scan(&lotteryCode, &typeID, &subID)
	if err != nil {
		t.Skipf("找不到可用的彩种 + 玩法：%v", err)
	}
	return &createEnv{
		svc: NewService(pool, nil), pool: pool, account: account,
		lottery: lotteryCode, playTypeID: typeID, subPlayID: subID,
	}
}

func (e *createEnv) create(t *testing.T, in CreateDefinitionInput) (Definition, error) {
	t.Helper()
	if in.SchemeName == "" {
		in.SchemeName = fmt.Sprintf("CREATE-TEST-%d", time.Now().UnixNano())
	}
	def, err := e.svc.CreateDefinition(context.Background(), e.account, in)
	if def.ID != "" {
		t.Cleanup(func() {
			_, _ = e.pool.Exec(context.Background(),
				`DELETE FROM scheme_definitions WHERE id = $1`, def.ID)
		})
	}
	return def, err
}

// TestCreateDefinitionRejectsBadCatalog 彩种或玩法对不上时必须在入口拦下。
// 这两条分支都要查库，纯函数校验覆盖不到。
func TestCreateDefinitionRejectsBadCatalog(t *testing.T) {
	env := newCreateEnv(t)

	t.Run("彩种不存在", func(t *testing.T) {
		_, err := env.create(t, CreateDefinitionInput{
			Kind: "custom", LotteryCode: "no_such_lottery_code",
			RunTypeID: "fixed_number", PlayTypeID: env.playTypeID, SubPlayID: env.subPlayID,
		})
		if !errors.Is(err, ErrInvalidCreateRequest) || !strings.Contains(err.Error(), "lotteryCode") {
			t.Fatalf("应报彩种无效，实际 %v", err)
		}
	})

	t.Run("玩法不存在", func(t *testing.T) {
		_, err := env.create(t, CreateDefinitionInput{
			Kind: "custom", LotteryCode: env.lottery,
			RunTypeID: "fixed_number", PlayTypeID: "g999", SubPlayID: "99999",
		})
		if !errors.Is(err, ErrInvalidCreateRequest) || !strings.Contains(err.Error(), "玩法") {
			t.Fatalf("应报玩法不存在，实际 %v", err)
		}
	})

	t.Run("会员不存在", func(t *testing.T) {
		_, err := env.svc.CreateDefinition(context.Background(), "no_such_account_zzz",
			CreateDefinitionInput{
				Kind: "custom", SchemeName: "CREATE-TEST-nomember", LotteryCode: env.lottery,
				RunTypeID: "fixed_number", PlayTypeID: env.playTypeID, SubPlayID: env.subPlayID,
			})
		if !errors.Is(err, member.ErrNotFound) {
			t.Fatalf("应报会员不存在，实际 %v", err)
		}
	})
}

// TestCreateDefinitionAllRunTypes 每种运行类型都要能建出方案，
// 且写进配置的彩种模板与玩法要与库里一致——后续出号、验奖、保存校验全靠这几个字段。
func TestCreateDefinitionAllRunTypes(t *testing.T) {
	env := newCreateEnv(t)

	var wantTemplate string
	if err := env.pool.QueryRow(context.Background(),
		`SELECT play_template FROM lottery_catalog WHERE code = $1`, env.lottery).Scan(&wantTemplate); err != nil {
		t.Fatalf("读彩种模板：%v", err)
	}

	for _, runType := range []string{
		RunTypeFixedNumber, RunTypeFixedRotate, RunTypeAdvFixedRotate,
		RunTypeAdvTriggerBet, RunTypeHotColdWarm, RunTypeRandomDraw,
	} {
		t.Run(runType, func(t *testing.T) {
			def, err := env.create(t, CreateDefinitionInput{
				Kind: "custom", LotteryCode: env.lottery, RunTypeID: runType,
				PlayTypeID: env.playTypeID, SubPlayID: env.subPlayID,
			})
			if err != nil {
				t.Fatalf("建方案失败：%v", err)
			}
			if def.ID == "" {
				t.Fatal("返回的方案 id 为空")
			}
			if def.LotteryCode != env.lottery {
				t.Fatalf("彩种 %s ≠ %s", def.LotteryCode, env.lottery)
			}
			if strings.TrimSpace(def.LotteryLabel) == "" {
				t.Error("彩种中文名未带出")
			}

			cfg := loadDefinitionConfig(t, env, def.ID)
			for key, want := range map[string]string{
				"runTypeId":    runType,
				"playTypeId":   env.playTypeID,
				"subPlayId":    env.subPlayID,
				"typeId":       env.playTypeID,
				"subId":        env.subPlayID,
				"playTemplate": wantTemplate,
				"lotteryCode":  env.lottery,
			} {
				if got, _ := cfg[key].(string); got != want {
					t.Errorf("配置 %s = %q，期望 %q", key, got, want)
				}
			}
		})
	}
}

// TestCreateDefinitionBuiltinPlanWithoutLottery 内置计划创建时不带彩种与玩法，
// 这些字段要等选了收藏方案才物化出来。
func TestCreateDefinitionBuiltinPlanWithoutLottery(t *testing.T) {
	env := newCreateEnv(t)
	def, err := env.create(t, CreateDefinitionInput{
		Kind: "custom", RunTypeID: RunTypeBuiltinPlan,
	})
	if err != nil {
		t.Fatalf("建内置计划方案失败：%v", err)
	}
	if def.LotteryCode != "" {
		t.Errorf("内置计划创建时不应带彩种，实际 %q", def.LotteryCode)
	}
	cfg := loadDefinitionConfig(t, env, def.ID)
	if got, _ := cfg["runTypeId"].(string); got != RunTypeBuiltinPlan {
		t.Errorf("配置 runTypeId = %q", got)
	}
	// 没有彩种就查不到模板，这一步应当被跳过而不是报错
	if _, ok := cfg["playTemplate"]; ok {
		t.Errorf("未选彩种时不应写入 playTemplate，实际 %v", cfg["playTemplate"])
	}
}

// TestCreateDefinitionRejectsDuplicateName 同一会员下方案重名要报专有错误，
// 而不是把库里的唯一约束冲突原样抛给调用方。
func TestCreateDefinitionRejectsDuplicateName(t *testing.T) {
	env := newCreateEnv(t)
	name := fmt.Sprintf("CREATE-TEST-dup-%d", time.Now().UnixNano())
	in := CreateDefinitionInput{
		Kind: "custom", SchemeName: name, LotteryCode: env.lottery,
		RunTypeID: RunTypeFixedNumber, PlayTypeID: env.playTypeID, SubPlayID: env.subPlayID,
	}
	if _, err := env.create(t, in); err != nil {
		t.Fatalf("首次创建失败：%v", err)
	}
	_, err := env.create(t, in)
	if !errors.Is(err, ErrNameDuplicate) {
		t.Fatalf("重名应报 ErrNameDuplicate，实际 %v", err)
	}
}

// TestCreateDefinitionDefaultsSchemeName 不给名字时应落一个默认名，而不是空串。
func TestCreateDefinitionDefaultsSchemeName(t *testing.T) {
	env := newCreateEnv(t)
	// 默认名固定为「新方案」，同名会撞唯一约束，所以先清掉可能的残留
	_, _ = env.pool.Exec(context.Background(),
		`DELETE FROM scheme_definitions WHERE scheme_name = '新方案'
		   AND member_id = (SELECT id FROM members WHERE account = $1)`, env.account)

	def, err := env.svc.CreateDefinition(context.Background(), env.account, CreateDefinitionInput{
		Kind: "custom", LotteryCode: env.lottery, RunTypeID: RunTypeFixedNumber,
		PlayTypeID: env.playTypeID, SubPlayID: env.subPlayID,
	})
	if err != nil {
		t.Fatalf("建方案失败：%v", err)
	}
	t.Cleanup(func() {
		_, _ = env.pool.Exec(context.Background(),
			`DELETE FROM scheme_definitions WHERE id = $1`, def.ID)
	})
	if def.SchemeName != "新方案" {
		t.Fatalf("默认方案名 = %q，期望「新方案」", def.SchemeName)
	}
}

func loadDefinitionConfig(t *testing.T, env *createEnv, defID string) map[string]interface{} {
	t.Helper()
	var raw []byte
	if err := env.pool.QueryRow(context.Background(),
		`SELECT config FROM scheme_definitions WHERE id = $1`, defID).Scan(&raw); err != nil {
		t.Fatalf("读方案配置：%v", err)
	}
	cfg := map[string]interface{}{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("解析方案配置：%v", err)
	}
	return cfg
}
