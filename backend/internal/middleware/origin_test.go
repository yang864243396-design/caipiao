package middleware

import "testing"

func TestIsAllowedOrigin(t *testing.T) {
	allowed := []string{"http://localhost:5173"}

	tests := []struct {
		name    string
		origin  string
		host    string
		want    bool
		allowed []string
	}{
		{name: "empty origin", origin: "", host: "example.com", want: true},
		{name: "whitelist", origin: "http://localhost:5173", host: "example.com", want: true},
		{name: "same host https", origin: "https://example.com", host: "example.com", want: true},
		{name: "same host http", origin: "http://example.com", host: "example.com", want: true},
		{name: "cross host", origin: "https://evil.com", host: "example.com", want: false},
		{name: "wildcard", origin: "https://any.test", host: "example.com", want: true, allowed: []string{"*"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origins := allowed
			if tt.allowed != nil {
				origins = tt.allowed
			}
			if got := IsAllowedOrigin(tt.origin, tt.host, origins); got != tt.want {
				t.Fatalf("IsAllowedOrigin(%q, %q) = %v, want %v", tt.origin, tt.host, got, tt.want)
			}
		})
	}
}
