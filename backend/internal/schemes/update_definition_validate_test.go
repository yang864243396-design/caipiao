package schemes

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func saveCfgJSON(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// contentPatch 一个"动了投注内容"的 patch，用于触发保存校验。
func contentPatch(groups ...string) UpdateDefinitionPatch {
	return UpdateDefinitionPatch{SchemeGroups: groups}
}

func TestValidateDefinitionContentOnSave(t *testing.T) {
	allDigits := "0,1,2,3,4,5,6,7,8,9"
	wuxingFushi := func(groups string) map[string]interface{} {
		return map[string]interface{}{
			"playTemplate": "ssc_std", "playTypeId": "g015", "subPlayId": "zhixuan_fs",
			"playMethod": "五星直选复式", "betMode": "fushi",
			"schemeGroups": []string{groups},
		}
	}

	cases := []struct {
		name      string
		kind      string
		cfg       map[string]interface{}
		patch     UpdateDefinitionPatch
		wantErr   bool
		wantMatch string
	}{
		{
			// 用户点名的用例：五星直选复式各位全选 = 10^5 = 100000 注，
			// 远超第三方上限，之前一路放行到下注才被拒。
			name: "全选五星直选复式应被拒绝",
			kind: "custom",
			cfg: wuxingFushi(strings.Join([]string{
				allDigits, allDigits, allDigits, allDigits, allDigits,
			}, "\n")),
			patch: contentPatch("x"), wantErr: true, wantMatch: "最大投注注数",
		},
		{
			name:  "五星直选复式各位只选一个号合法",
			kind:  "custom",
			cfg:   wuxingFushi("1\n2\n3\n4\n5"),
			patch: contentPatch("x"), wantErr: false,
		},
		{
			// 各位同号 → 第三方注数为 0，下单必被丢弃
			name:  "五星直选复式豹子应被拒绝",
			kind:  "custom",
			cfg:   wuxingFushi("7\n7\n7\n7\n7"),
			patch: contentPatch("x"), wantErr: true, wantMatch: "注数为 0",
		},
		{
			// 库里真实存在的坏配置：龙虎玩法却填了数字
			name: "龙虎玩法填数字应被拒绝",
			kind: "custom",
			cfg: map[string]interface{}{
				"playTemplate": "ssc_std", "playTypeId": "longhu", "subPlayId": "lh_wan_ge",
				"playMethod": "龙虎", "betMode": "longhu", "schemeGroups": []string{"6"},
			},
			patch: contentPatch("x"), wantErr: true, wantMatch: "号池",
		},
		{
			// 和值号池是 0..27，内容写成补零的 04 是同一个值，不能报越界
			name: "和值补零写法不应误拦",
			kind: "custom",
			cfg: map[string]interface{}{
				"playTemplate": "pc28_std", "playTypeId": "hezhi", "subPlayId": "hz",
				"playMethod": "和值", "betMode": "hezhi", "schemeGroups": []string{"04,07,27"},
			},
			patch: contentPatch("x"), wantErr: false,
		},
		{
			// 任选内容前半段是位名不是号码，剥不掉就会把正常方案全误伤
			name: "任选位名前缀不应误拦",
			kind: "custom",
			cfg: map[string]interface{}{
				"playTemplate": "ssc_std", "playTypeId": "renxuan", "subPlayId": "ren2_zhixuan_ds",
				"playMethod": "任选二直选单式", "betMode": "danshi",
				"schemeGroups": []string{"千,十|12,34"},
			},
			patch: contentPatch("x"), wantErr: false,
		},
		{
			// 只改方案名时不该被历史遗留的非法配置挡住——用户既看不懂也修不了
			name:  "未触及投注内容的 patch 不校验",
			kind:  "custom",
			cfg:   wuxingFushi("7\n7\n7\n7\n7"),
			patch: UpdateDefinitionPatch{HasSchemeName: true, SchemeName: "改个名"},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDefinitionContentOnSave(tc.kind, saveCfgJSON(t, tc.cfg), tc.patch)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("不应拦截，却报了：%v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("应被拦截，却放行了")
			}
			if !errors.Is(err, ErrInvalidSchemeContent) {
				t.Fatalf("错误类型应为 ErrInvalidSchemeContent：%v", err)
			}
			if !strings.Contains(err.Error(), tc.wantMatch) {
				t.Fatalf("错误信息应含 %q：%v", tc.wantMatch, err)
			}
		})
	}
}
