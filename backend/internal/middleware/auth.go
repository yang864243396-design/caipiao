package middleware

import (
	"context"
	"net/http"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/auth"
)

type ctxKey int

const claimsKey ctxKey = 1

func WithClaims(ctx context.Context, c auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(auth.Claims)
	return c, ok
}

func RequireRole(svc *auth.Service, role auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				apix.Unauthorized(w, "")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			claims, err := svc.ParseBearer(token)
			if err != nil {
				apix.Unauthorized(w, "")
				return
			}
			if claims.Role != role {
				apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "无权访问")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}
