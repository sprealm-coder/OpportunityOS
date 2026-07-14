package application

import (
	"context"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/finance"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	orderdomain "github.com/opportunity-os/opportunity-os/services/core-api/internal/order"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/pricing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/routing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

type BlueprintInput struct {
	Name                   string
	Description            string
	ValueProposition       string
	RequiredCapabilities   []string
	ProductDefinitions     []map[string]any
	WorkflowDefinitions    []map[string]any
	MeteringDefinitions    []map[string]any
	PricingDefinitions     []map[string]any
	ComplianceProfile      map[string]any
	BusinessModel          map[string]any
	TargetMarketDefinition map[string]any
	RevenueModel           map[string]any
	DeliveryModel          map[string]any
}

type ProductVersionInput struct {
	InputSchema       schema.Definition          `json:"input_schema"`
	OutputSchema      schema.Definition          `json:"output_schema"`
	FormSchema        schema.Definition          `json:"form_schema"`
	CapabilityIDs     []string                   `json:"capability_ids"`
	Workflow          workflow.Definition        `json:"workflow"`
	Metering          pricing.MeteringDefinition `json:"metering"`
	PriceBook         pricing.PriceBook          `json:"price_book"`
	RoutePolicy       routing.Policy             `json:"route_policy"`
	DeliveryMode      string                     `json:"delivery_mode"`
	ComplianceProfile map[string]any             `json:"compliance_profile"`
	GrowthPlaybook    map[string]any             `json:"growth_playbook"`
}

type SKUVersionInput struct {
	ProductVersionID string         `json:"product_version_id"`
	Entitlements     map[string]any `json:"entitlements"`
}

type QuoteItemInput struct {
	SKUVersionID string         `json:"sku_version_id"`
	Quantity     int64          `json:"quantity"`
	Input        map[string]any `json:"input"`
}

type QuoteInput struct {
	DealID     string           `json:"deal_id"`
	CustomerID string           `json:"customer_id"`
	Currency   string           `json:"currency"`
	ValidUntil time.Time        `json:"valid_until"`
	Items      []QuoteItemInput `json:"items"`
}

type ExecutionTransitionInput struct {
	To                 string         `json:"to"`
	ProviderEndpointID string         `json:"provider_endpoint_id"`
	ExternalID         string         `json:"external_id"`
	Output             map[string]any `json:"output"`
	Error              map[string]any `json:"error"`
}

type WalletInput struct {
	OwnerType string `json:"owner_type"`
	OwnerID   string `json:"owner_id"`
	Currency  string `json:"currency"`
}

type WalletAdjustmentInput struct {
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}

type HoldInput struct {
	WalletID    string `json:"wallet_id"`
	AmountMinor int64  `json:"amount_minor"`
}

type ReleaseInput struct {
	AmountMinor int64 `json:"amount_minor"`
}

type ChargePostingInput struct {
	HoldID string `json:"hold_id"`
}

type RefundInput struct {
	WalletID    string `json:"wallet_id"`
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}

type CommissionInput struct {
	BeneficiaryType string `json:"beneficiary_type"`
	BeneficiaryID   string `json:"beneficiary_id"`
	AmountMinor     int64  `json:"amount_minor"`
}

type SettlementInput struct {
	SourceType  string `json:"source_type"`
	SourceID    string `json:"source_id"`
	AmountMinor int64  `json:"amount_minor"`
}

type ReconciliationInput struct {
	OrderID string `json:"order_id"`
}

type Store interface {
	CreateSession(context.Context, string, string) (auth.Session, error)
	ResolveSession(context.Context, string) (auth.Session, error)
	RevokeSession(context.Context, string) error

	CreateOpportunity(context.Context, tenancy.Scope, string, string, string) (opportunity.Opportunity, error)
	ListOpportunities(context.Context, tenancy.Scope) ([]opportunity.Opportunity, error)
	GetOpportunity(context.Context, tenancy.Scope, string) (opportunity.Opportunity, error)
	AddEvidence(context.Context, tenancy.Scope, string, opportunity.Evidence, string) (opportunity.Opportunity, error)
	ScoreOpportunity(context.Context, tenancy.Scope, string, int, string) (opportunity.Opportunity, error)
	TransitionOpportunity(context.Context, tenancy.Scope, string, string, string) (opportunity.Opportunity, error)
	ReviewOpportunity(context.Context, tenancy.Scope, string, string, string, string) (opportunity.Opportunity, error)
	ListAudit(context.Context, tenancy.Scope) ([]audit.Record, error)

	CreateIncubation(context.Context, tenancy.Scope, string, string, string) (incubation.Project, error)
	ListIncubations(context.Context, tenancy.Scope) ([]incubation.Project, error)
	TransitionIncubation(context.Context, tenancy.Scope, string, string, string) (incubation.Project, error)

	CreateBlueprint(context.Context, tenancy.Scope, string, BlueprintInput, string) (blueprint.BusinessBlueprint, error)
	ListBlueprints(context.Context, tenancy.Scope) ([]blueprint.BusinessBlueprint, error)
	TransitionBlueprint(context.Context, tenancy.Scope, string, string, string) (blueprint.BusinessBlueprint, error)

	CreateCapability(context.Context, tenancy.Scope, string, string, map[string]any, string) (capability.Capability, error)
	ListCapabilities(context.Context, tenancy.Scope) ([]capability.Capability, error)
	CreateProvider(context.Context, tenancy.Scope, string, string) (capability.Provider, error)
	ListProviders(context.Context, tenancy.Scope) ([]capability.Provider, error)
	CreateProviderEndpoint(context.Context, tenancy.Scope, string, string, string, string, string) (capability.ProviderEndpoint, error)

	CreateProduct(context.Context, tenancy.Scope, string, string, string) (catalog.Product, error)
	ListProducts(context.Context, tenancy.Scope) ([]catalog.Product, error)
	GetProduct(context.Context, tenancy.Scope, string) (catalog.ProductDetail, error)
	CreateProductVersion(context.Context, tenancy.Scope, string, ProductVersionInput, string) (catalog.ProductVersion, error)
	CreateSKU(context.Context, tenancy.Scope, string, string, string, string) (catalog.SKU, error)
	CreateSKUVersion(context.Context, tenancy.Scope, string, SKUVersionInput, string) (catalog.SKUVersion, error)
	PublishProduct(context.Context, tenancy.Scope, string, string, string) (catalog.Publication, error)
}

type TransactionStore interface {
	CreateQuote(context.Context, tenancy.Scope, QuoteInput, string) (orderdomain.Quote, error)
	ListQuotes(context.Context, tenancy.Scope) ([]orderdomain.Quote, error)
	GetQuote(context.Context, tenancy.Scope, string) (orderdomain.Quote, error)
	TransitionQuote(context.Context, tenancy.Scope, string, string, string) (orderdomain.Quote, error)

	CreateOrder(context.Context, tenancy.Scope, string, string) (orderdomain.Order, error)
	ListOrders(context.Context, tenancy.Scope) ([]orderdomain.Order, error)
	GetOrder(context.Context, tenancy.Scope, string) (orderdomain.Order, error)
	TransitionOrder(context.Context, tenancy.Scope, string, string, string) (orderdomain.Order, error)
	TransitionExecution(context.Context, tenancy.Scope, string, ExecutionTransitionInput, string) (orderdomain.ExecutionOrder, error)
	TransitionDelivery(context.Context, tenancy.Scope, string, string, string) (orderdomain.DeliveryProject, error)
	RecordUsage(context.Context, tenancy.Scope, string, int64, time.Time, string) (orderdomain.UsageRecord, error)
	RecordProviderCost(context.Context, tenancy.Scope, string, string, string, int64, string) (orderdomain.ProviderCost, error)
	CreateCustomerCharge(context.Context, tenancy.Scope, string, string) (orderdomain.CustomerCharge, error)
}

type FinanceStore interface {
	ListFinance(context.Context, tenancy.Scope) (finance.Overview, error)
	CreateWallet(context.Context, tenancy.Scope, WalletInput, string) (finance.Wallet, error)
	PostWalletAdjustment(context.Context, tenancy.Scope, string, WalletAdjustmentInput, string) (finance.Adjustment, error)
	PlaceOrderHold(context.Context, tenancy.Scope, string, HoldInput, string) (finance.Hold, error)
	ReleaseHold(context.Context, tenancy.Scope, string, ReleaseInput, string) (finance.Hold, error)
	PostCustomerCharge(context.Context, tenancy.Scope, string, ChargePostingInput, string) (finance.Transaction, error)
	RefundCustomerCharge(context.Context, tenancy.Scope, string, RefundInput, string) (finance.Refund, error)
	CreateCommission(context.Context, tenancy.Scope, string, CommissionInput, string) (finance.Commission, error)
	CreateProviderPayable(context.Context, tenancy.Scope, string, string) (finance.ProviderPayable, error)
	CreateSettlement(context.Context, tenancy.Scope, SettlementInput, string) (finance.Settlement, error)
	RunReconciliation(context.Context, tenancy.Scope, ReconciliationInput, string) (finance.ReconciliationRun, error)
}
