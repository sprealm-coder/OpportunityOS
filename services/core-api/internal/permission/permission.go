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
	TransactionRead     = "transaction.read"
	QuoteWrite          = "quote.write"
	OrderWrite          = "order.write"
	ExecutionWrite      = "execution.write"
	BillingWrite        = "billing.write"
	FinanceRead         = "finance.read"
	WalletWrite         = "wallet.write"
	FinanceAdjust       = "finance.adjust"
	LedgerPost          = "ledger.post"
	SettlementWrite     = "settlement.write"
	ReconciliationWrite = "reconciliation.write"
	GrowthRead          = "growth.read"
	GrowthWrite         = "growth.write"
	LeadWrite           = "lead.write"
	ProofWrite          = "proof.write"
	ProofReview         = "proof.review"
	CampaignWrite       = "campaign.write"
	CampaignApprove     = "campaign.approve"
	OutreachWrite       = "outreach.write"
	DealWrite           = "deal.write"
	ExperimentWrite     = "experiment.write"
	ChannelRead         = "channel.read"
	ResellerWrite       = "reseller.write"
	OwnershipWrite      = "ownership.write"
	OwnershipApprove    = "ownership.approve"
	SupplierWrite       = "supplier.write"
	SupplierApprove     = "supplier.approve"
	MarketplaceWrite    = "marketplace.write"
	MarketplaceReview   = "marketplace.review"
	MarketplaceTakedown = "marketplace.takedown"
)

var rolePermissions = map[string]map[string]bool{
	"admin": {"*": true},
	"operator": {
		OpportunityRead: true, OpportunityCreate: true, OpportunityEvidence: true,
		OpportunityScore: true, OpportunitySubmit: true, IncubationRead: true,
		IncubationWrite: true, BlueprintRead: true, BlueprintWrite: true,
		CapabilityRead: true, CapabilityWrite: true, ProviderRead: true, ProviderWrite: true,
		ProductRead: true, ProductWrite: true, ProductPublish: true,
		TransactionRead: true, QuoteWrite: true, OrderWrite: true,
		ExecutionWrite: true, BillingWrite: true,
		FinanceRead: true, WalletWrite: true, LedgerPost: true,
		SettlementWrite: true, ReconciliationWrite: true,
		GrowthRead: true, GrowthWrite: true, LeadWrite: true, ProofWrite: true,
		CampaignWrite: true, OutreachWrite: true, DealWrite: true, ExperimentWrite: true,
		ChannelRead: true, ResellerWrite: true, OwnershipWrite: true,
		SupplierWrite: true, MarketplaceWrite: true,
	},
	"reviewer": {OpportunityRead: true, OpportunityReview: true, CapabilityRead: true, ProviderRead: true, ProductRead: true, TransactionRead: true, FinanceRead: true, GrowthRead: true, ProofReview: true, CampaignApprove: true, ChannelRead: true, OwnershipApprove: true, SupplierApprove: true, MarketplaceReview: true, MarketplaceTakedown: true, AuditRead: true},
	"auditor":  {OpportunityRead: true, IncubationRead: true, BlueprintRead: true, CapabilityRead: true, ProviderRead: true, ProductRead: true, TransactionRead: true, FinanceRead: true, GrowthRead: true, ChannelRead: true, AuditRead: true},
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
