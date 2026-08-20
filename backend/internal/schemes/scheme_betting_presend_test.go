package schemes

import "testing"

func TestProviderChangedPeriodFromPreSendError(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		want   string
	}{
		{
			name:   "wrapped verifier error",
			detail: "provider period verification failed (verify_ms=0): provider open period changed from 10114252803595 to 10114252803596",
			want:   "10114252803596",
		},
		{
			name:   "preserves original error before deferred suffix",
			detail: "provider period verification failed (verify_ms=0): provider open period changed from P100 to P101; pre_send_reschedule_deferred=no fresh provider target for pre-send replacement",
			want:   "P101",
		},
		{name: "other failure", detail: "provider period is inside the dispatch safety margin"},
		{name: "missing target", detail: "provider open period changed from P100 to "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := providerChangedPeriodFromPreSendError(tt.detail); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
