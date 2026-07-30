package guaji

import "testing"

// WebSocket 握手不跟随重定向。2026-07-28 上游把 /ws 改成 301 → /ws/，
// 开奖 WS 全线 bad handshake、所有彩种停止入库，探针只报「不可达」查不出原因。
func TestWSRedirectTarget(t *testing.T) {
	const src = "wss://www.v6hs1.com/ws?token=Anonymous"

	cases := []struct {
		name     string
		raw      string
		location string
		want     string
	}{
		{
			// 实测响应：Location 是 http:// 明文，不能把 wss 降级。
			name:     "补尾斜杠并保持 wss",
			raw:      src,
			location: "http://www.v6hs1.com/ws/",
			want:     "wss://www.v6hs1.com/ws/?token=Anonymous",
		},
		{
			name:     "相对路径",
			raw:      src,
			location: "/ws2/",
			want:     "wss://www.v6hs1.com/ws2/?token=Anonymous",
		},
		{
			name:     "Location 自带查询串时不覆盖",
			raw:      src,
			location: "https://www.v6hs1.com/ws/?token=X",
			want:     "wss://www.v6hs1.com/ws/?token=X",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := wsRedirectTarget(tc.raw, tc.location)
			if err != nil {
				t.Fatalf("解析 Location 失败: %v", err)
			}
			if got != tc.want {
				t.Fatalf("目标 = %q，期望 %q", got, tc.want)
			}
		})
	}
}

// 无法得出新地址时必须报错，否则会拿原地址空转重试。
func TestWSRedirectTargetRejectsUseless(t *testing.T) {
	const src = "wss://www.v6hs1.com/ws/?token=Anonymous"
	for _, loc := range []string{"", "   ", "wss://www.v6hs1.com/ws/?token=Anonymous", "mailto:x@y.z"} {
		if got, err := wsRedirectTarget(src, loc); err == nil {
			t.Fatalf("Location %q 应报错，却给出 %q", loc, got)
		}
	}
}

// 默认路径必须带尾斜杠，否则线上又会踩 301。
func TestWSPathDefaultHasTrailingSlash(t *testing.T) {
	if got := wsPathOrDefault(""); got != "/ws/" {
		t.Fatalf("默认 WS 路径 = %q，期望 /ws/", got)
	}
	if got := wsPathOrDefault("ws2"); got != "/ws2" {
		t.Fatalf("显式配置应原样带上前导斜杠，得到 %q", got)
	}
}
