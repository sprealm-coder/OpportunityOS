package permission

import (
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"sync"
)

type Grant struct{ TenantID, ActorID, Permission string }
type Authorizer struct {
	mu     sync.RWMutex
	grants map[string]bool
}

func New() *Authorizer               { return &Authorizer{grants: map[string]bool{}} }
func grantKey(t, a, p string) string { return t + "/" + a + "/" + p }
func (a *Authorizer) Grant(grant Grant) error {
	if grant.TenantID == "" {
		return platform.ErrTenantRequired
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.grants[grantKey(grant.TenantID, grant.ActorID, grant.Permission)] = true
	return nil
}
func (a *Authorizer) Require(tenantID, actorID, permission string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.grants[grantKey(tenantID, actorID, permission)] {
		return platform.ErrPermissionDenied
	}
	return nil
}
