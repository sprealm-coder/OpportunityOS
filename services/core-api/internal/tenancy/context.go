package tenancy

import (
	"context"
	"net/http"
	"strings"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type contextKey string

const scopeKey contextKey = "tenant-scope"

type Scope struct {
	TenantID string
	ActorID  string
	TraceID  string
	Role     string
}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeKey, scope)
}

func FromContext(ctx context.Context) (Scope, error) {
	scope, ok := ctx.Value(scopeKey).(Scope)
	if !ok || scope.TenantID == "" {
		return Scope{}, platform.ErrTenantRequired
	}
	return scope, nil
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if tenantID == "" || actorID == "" {
			http.Error(w, "tenant and actor headers are required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithScope(r.Context(), Scope{TenantID: tenantID, ActorID: actorID, TraceID: r.Header.Get("X-Trace-ID")})))
	})
}

func CacheKey(tenantID, namespace, key string) (string, error) {
	if tenantID == "" {
		return "", platform.ErrTenantRequired
	}
	return "tenant:" + tenantID + ":" + namespace + ":" + key, nil
}
