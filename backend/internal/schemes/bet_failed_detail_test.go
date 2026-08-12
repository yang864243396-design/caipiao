package schemes

import "testing"

func TestEnsureBetFailedDetail_EmptyPreflightFailureIsExplainable(t *testing.T) {
	const want = "本地预检失败，第三方投注请求未发送"
	if got := ensureBetFailedDetail(""); got != want {
		t.Fatalf("ensureBetFailedDetail(empty) = %q, want %q", got, want)
	}
}

func TestEnsureBetFailedDetail_PreservesUpstreamReason(t *testing.T) {
	if got := ensureBetFailedDetail("投注注数不正确"); got != "投注注数不正确" {
		t.Fatalf("ensureBetFailedDetail = %q", got)
	}
}
