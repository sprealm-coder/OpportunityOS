package permission

import (
	"sync"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

const (
	OpportunityRead     = "opportunity.read"
	OpportunityCreate   = "opportunity.create"
	OpportunityEvidence = "opportunity.evidence"
	OpportunityScore    = "opportunity.score"
	OpportunitySubmit   = "opportunity.submit_review"
	OpportunityReview   = "opportunity.review"
	IncubationRead      = "incubation.read"
	IncubationWrite     = "incubation.write"
	BlueprintRead       = "blueprint.read"
	BlueprintWrite      = "blueprint.write"
	AuditRead           = "audit.read"
	CapabilityRead      = "capability.read"
	CapabilityWrite     = "capability.write"
	ProviderRead        = "provider.read"
	ProviderWrite       = "provider.write"
	ProductRead         = "product.read"
	ProductWrite        = "product.write"
	ProductPublish      = "product.publish"
)

var rolePermissions = map[string]map[string]bool{
	"admin": {"*": true},
	"operator": {
		OpportunityRead: true, OpportunityCreate: true, OpportunityEvidence: true,
		OpportunityScore: true, OpportunitySubmit: true, IncubationRead: true,
		IncubationWrite: true, BlueprintRead: true, BlueprintWrite: true,
		CapabilityRead: true, CapabilityWrite: true, ProviderRead: true, ProviderWrite: true,
		ProductRead: true, ProductWrite: true, ProductPublish: true,
	},
	"reviewer": {OpportunityRead: true, OpportunityReview: true, ProductRead: true, AuditRead: true},
	"auditor":  {OpportunityRead: true, IncubationRead: true, BlueprintRead: true, CapabilityRead: true, ProviderRead: true, ProductRead: true, AuditRead: true},
}

func RequireRole(role, required string) error {
	permissions := rolePermissions[role]
	if permissions == nil || (!permissions["*"] && !permissions[required]) {
		return platform.ErrPermissionDenied
	}
	return nil
}

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
