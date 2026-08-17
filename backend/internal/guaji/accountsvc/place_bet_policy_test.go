package accountsvc

import (
	"errors"
	"testing"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

func TestPlaceLottBetBusinessError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"insufficient", &guaji.APIError{Code: 40000, Message: "余额不足"}, guajibet.ErrInsufficient},
		{"closed", &guaji.APIError{Code: 40000, Message: "已过投注截止时间"}, guajibet.ErrPeriodClosed},
		{"amount", &guaji.APIError{Code: 40000, Message: "投注金额不正确"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := placeLottBetBusinessError(tc.err)
			if tc.want == nil && got != nil {
				t.Fatalf("got %v want nil", got)
			}
			if tc.want != nil && !errors.Is(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
