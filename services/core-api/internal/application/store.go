package application

import (
	"context"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/channel"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/finance"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/growth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/marketplace"
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

type MarketSegmentInput struct {
	Name       string         `json:"name"`
	Definition map[string]any `json:"definition"`
}

type ICPDefinitionInput struct {
	Name       string         `json:"name"`
	Definition map[string]any `json:"definition"`
}

type LeadInput struct {
	MarketSegmentID string `json:"market_segment_id"`
	ICPDefinitionID string `json:"icp_definition_id"`
	Name            string `json:"name"`
}

type LeadEvidenceInput struct {
	Kind       string `json:"kind"`
	Summary    string `json:"summary"`
	Confidence int    `json:"confidence"`
	SourceRef  string `json:"source_ref"`
}

type ContactInput struct {
	Channel       string         `json:"channel"`
	Value         string         `json:"value"`
	ConsentStatus string         `json:"consent_status"`
	SourceType    string         `json:"source_type"`
	SourceRef     string         `json:"source_ref"`
	Evidence      map[string]any `json:"evidence"`
}

type ProofTemplateInput struct {
	Name              string            `json:"name"`
	ProofType         string            `json:"proof_type"`
	WorkflowVersionID string            `json:"workflow_version_id"`
	InputSchema       schema.Definition `json:"input_schema"`
	OutputSchema      schema.Definition `json:"output_schema"`
	AccessPolicy      map[string]any    `json:"access_policy"`
	RetentionDays     int               `json:"retention_days"`
}

type ProofRequestInput struct {
	TemplateID string         `json:"template_id"`
	DealID     string         `json:"deal_id"`
	Input      map[string]any `json:"input"`
	ExpiresAt  time.Time      `json:"expires_at"`
}

type ProofGenerationInput struct {
	Result      map[string]any `json:"result"`
	ArtifactRef string         `json:"artifact_ref"`
}

type ProofReviewInput struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

type CampaignInput struct {
	MarketSegmentID string `json:"market_segment_id"`
	Name            string `json:"name"`
	Channel         string `json:"channel"`
	Purpose         string `json:"purpose"`
}

type CampaignStepInput struct {
	Position   int            `json:"position"`
	Kind       string         `json:"kind"`
	Definition map[string]any `json:"definition"`
}

type CampaignApprovalInput struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

type SuppressionInput struct {
	LeadID    string `json:"lead_id"`
	ContactID string `json:"contact_id"`
	Channel   string `json:"channel"`
	Reason    string `json:"reason"`
	SourceRef string `json:"source_ref"`
}

type OutreachPlanInput struct {
	LeadID    string         `json:"lead_id"`
	StepID    string         `json:"step_id"`
	ContactID string         `json:"contact_id"`
	Content   map[string]any `json:"content"`
}

type OutreachTransitionInput struct {
	To                string `json:"to"`
	ExternalMessageID string `json:"external_message_id"`
}

type ConversationInput struct {
	LeadID  string `json:"lead_id"`
	DealID  string `json:"deal_id"`
	Channel string `json:"channel"`
}

type ConversationMessageInput struct {
	Direction string         `json:"direction"`
	Status    string         `json:"status"`
	Content   map[string]any `json:"content"`
}

type DealInput struct {
	LeadID     string `json:"lead_id"`
	Name       string `json:"name"`
	CustomerID string `json:"customer_id"`
	Currency   string `json:"currency"`
	ValueMinor int64  `json:"value_minor"`
}

type ExperimentInput struct {
	Name                  string         `json:"name"`
	EntityType            string         `json:"entity_type"`
	EntityID              string         `json:"entity_id"`
	Hypothesis            string         `json:"hypothesis"`
	AllocationBasisPoints int            `json:"allocation_basis_points"`
	MetricsDefinition     map[string]any `json:"metrics_definition"`
}

type ExperimentTransitionInput struct {
	To     string         `json:"to"`
	Result map[string]any `json:"result"`
}

type ResellerLevelInput struct {
	Name                 string `json:"name"`
	Rank                 int    `json:"rank"`
	DefaultCommissionBPS int    `json:"default_commission_bps"`
}

type ResellerInput struct {
	LevelID string `json:"level_id"`
	Name    string `json:"name"`
}

type AttributionRuleInput struct {
	Name       string         `json:"name"`
	Priority   int            `json:"priority"`
	Definition map[string]any `json:"definition"`
}

type LeadOwnershipInput struct {
	LeadID            string `json:"lead_id"`
	ResellerID        string `json:"reseller_id"`
	AttributionRuleID string `json:"attribution_rule_id"`
	ProtectionDays    int    `json:"protection_days"`
}

type CustomerOwnershipInput struct {
	CustomerID            string `json:"customer_id"`
	ResellerID            string `json:"reseller_id"`
	SourceLeadOwnershipID string `json:"source_lead_ownership_id"`
	ProtectionDays        int    `json:"protection_days"`
}

type TransferRequestInput struct {
	OwnershipType string `json:"ownership_type"`
	OwnershipID   string `json:"ownership_id"`
	ToResellerID  string `json:"to_reseller_id"`
	Rationale     string `json:"rationale"`
}

type ReviewInput struct {
	Decision  string `json:"decision"`
	Rationale string `json:"rationale"`
}

type CommissionRuleInput struct {
	Name            string     `json:"name"`
	ResellerID      string     `json:"reseller_id"`
	ResellerLevelID string     `json:"reseller_level_id"`
	BasisPoints     int        `json:"basis_points"`
	EffectiveFrom   time.Time  `json:"effective_from"`
	EffectiveUntil  *time.Time `json:"effective_until"`
}

type CommissionLockInput struct {
	CommissionID     string `json:"commission_id"`
	CommissionRuleID string `json:"commission_rule_id"`
	ResellerID       string `json:"reseller_id"`
}

type SettlementCycleInput struct {
	ResellerID  string    `json:"reseller_id"`
	Name        string    `json:"name"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

type SupplierInput struct {
	Name string `json:"name"`
}

type SupplierCapabilityInput struct {
	SupplierID   string `json:"supplier_id"`
	CapabilityID string `json:"capability_id"`
}

type ProviderSupplierInput struct {
	SupplierID string `json:"supplier_id"`
}

type SupplierContractInput struct {
	SupplierID string         `json:"supplier_id"`
	ProviderID string         `json:"provider_id"`
	Name       string         `json:"name"`
	Currency   string         `json:"currency"`
	Terms      map[string]any `json:"terms"`
	StartsAt   *time.Time     `json:"starts_at"`
	EndsAt     *time.Time     `json:"ends_at"`
}

type SupplierRateInput struct {
	ContractID   string `json:"contract_id"`
	CapabilityID string `json:"capability_id"`
	Unit         string `json:"unit"`
	RateMinor    int64  `json:"rate_minor"`
}

type SupplierQualityInput struct {
	SupplierID         string         `json:"supplier_id"`
	ProviderID         string         `json:"provider_id"`
	ProviderEndpointID string         `json:"provider_endpoint_id"`
	Metric             string         `json:"metric"`
	ScoreBPS           int            `json:"score_bps"`
	Evidence           map[string]any `json:"evidence"`
	PeriodStart        time.Time      `json:"period_start"`
	PeriodEnd          time.Time      `json:"period_end"`
}

type DeveloperInput struct {
	Name string `json:"name"`
}

type PublisherInput struct {
	DeveloperID string `json:"developer_id"`
	Name        string `json:"name"`
}

type ListingInput struct {
	PublisherID string `json:"publisher_id"`
	Name        string `json:"name"`
	ListingType string `json:"listing_type"`
}
type ListingVersionInput struct {
	CapabilityManifest map[string]any `json:"capability_manifest"`
	PermissionManifest map[string]any `json:"permission_manifest"`
	ContentRef         string         `json:"content_ref"`
	Checksum           string         `json:"checksum"`
}
type ListingReviewInput struct {
	ListingVersionID string `json:"listing_version_id"`
	ReviewType       string `json:"review_type"`
	Decision         string `json:"decision"`
	Rationale        string `json:"rationale"`
}
type SandboxRunInput struct {
	ListingVersionID string         `json:"listing_version_id"`
	Status           string         `json:"status"`
	Policy           map[string]any `json:"policy"`
	Result           map[string]any `json:"result"`
}
type ListingQualityInput struct {
	ListingVersionID string         `json:"listing_version_id"`
	ScoreBPS         int            `json:"score_bps"`
	Dimensions       map[string]any `json:"dimensions"`
}
type MarketplaceDisputeInput struct {
	ListingID    string `json:"listing_id"`
	ClaimantType string `json:"claimant_type"`
	ClaimantID   string `json:"claimant_id"`
	Reason       string `json:"reason"`
}

type DisputeResolutionInput struct {
	Decision   string         `json:"decision"`
	Resolution map[string]any `json:"resolution"`
}

type TakedownInput struct {
	ListingID string `json:"listing_id"`
	Reason    string `json:"reason"`
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

type GrowthStore interface {
	ListGrowth(context.Context, tenancy.Scope) (growth.Overview, error)
	CreateMarketSegment(context.Context, tenancy.Scope, MarketSegmentInput, string) (growth.MarketSegment, error)
	CreateICPDefinition(context.Context, tenancy.Scope, string, ICPDefinitionInput, string) (growth.ICPDefinition, error)
	CreateLead(context.Context, tenancy.Scope, LeadInput, string) (growth.Lead, error)
	AddLeadEvidence(context.Context, tenancy.Scope, string, LeadEvidenceInput, string) (growth.LeadEvidence, error)
	TransitionLead(context.Context, tenancy.Scope, string, string, string) (growth.Lead, error)
	CreateContact(context.Context, tenancy.Scope, string, ContactInput, string) (growth.Contact, error)
	CreateProofTemplate(context.Context, tenancy.Scope, ProofTemplateInput, string) (growth.ProofTemplate, error)
	CreateProofRequest(context.Context, tenancy.Scope, string, ProofRequestInput, string) (growth.ProofRequest, error)
	GenerateProof(context.Context, tenancy.Scope, string, ProofGenerationInput, string) (growth.ProofInstance, error)
	ReviewProof(context.Context, tenancy.Scope, string, ProofReviewInput, string) (growth.ProofInstance, error)
	CreateCampaign(context.Context, tenancy.Scope, CampaignInput, string) (growth.Campaign, error)
	AddCampaignStep(context.Context, tenancy.Scope, string, CampaignStepInput, string) (growth.CampaignStep, error)
	TransitionCampaign(context.Context, tenancy.Scope, string, string, string) (growth.Campaign, error)
	ReviewCampaign(context.Context, tenancy.Scope, string, CampaignApprovalInput, string) (growth.Campaign, error)
	CreateSuppression(context.Context, tenancy.Scope, SuppressionInput, string) (growth.SuppressionEntry, error)
	PlanOutreach(context.Context, tenancy.Scope, string, OutreachPlanInput, string) (growth.OutreachMessage, error)
	TransitionOutreach(context.Context, tenancy.Scope, string, OutreachTransitionInput, string) (growth.OutreachMessage, error)
	CreateConversation(context.Context, tenancy.Scope, ConversationInput, string) (growth.Conversation, error)
	AddConversationMessage(context.Context, tenancy.Scope, string, ConversationMessageInput, string) (growth.ConversationMessage, error)
	CreateDeal(context.Context, tenancy.Scope, DealInput, string) (growth.Deal, error)
	GetDeal(context.Context, tenancy.Scope, string) (growth.Deal, error)
	TransitionDeal(context.Context, tenancy.Scope, string, string, string) (growth.Deal, error)
	CreateExperiment(context.Context, tenancy.Scope, ExperimentInput, string) (growth.Experiment, error)
	TransitionExperiment(context.Context, tenancy.Scope, string, ExperimentTransitionInput, string) (growth.Experiment, error)
}

type ChannelStore interface {
	ListChannels(context.Context, tenancy.Scope) (channel.Overview, error)
	CreateResellerLevel(context.Context, tenancy.Scope, ResellerLevelInput, string) (channel.ResellerLevel, error)
	CreateReseller(context.Context, tenancy.Scope, ResellerInput, string) (channel.Reseller, error)
	CreateAttributionRule(context.Context, tenancy.Scope, AttributionRuleInput, string) (channel.AttributionRule, error)
	AssignLeadOwnership(context.Context, tenancy.Scope, LeadOwnershipInput, string) (channel.LeadOwnership, error)
	CreateCustomerOwnership(context.Context, tenancy.Scope, CustomerOwnershipInput, string) (channel.CustomerOwnership, error)
	CreateTransferRequest(context.Context, tenancy.Scope, TransferRequestInput, string) (channel.TransferRequest, error)
	ReviewTransfer(context.Context, tenancy.Scope, string, ReviewInput, string) (channel.TransferRequest, error)
	CreateCommissionRule(context.Context, tenancy.Scope, CommissionRuleInput, string) (channel.CommissionRule, error)
	LockCommission(context.Context, tenancy.Scope, CommissionLockInput, string) (channel.CommissionLock, error)
	CreateSettlementCycle(context.Context, tenancy.Scope, SettlementCycleInput, string) (channel.SettlementCycle, error)
	CreateSupplier(context.Context, tenancy.Scope, SupplierInput, string) (channel.Supplier, error)
	BindSupplierCapability(context.Context, tenancy.Scope, SupplierCapabilityInput, string) (channel.SupplierCapability, error)
	BindProviderSupplier(context.Context, tenancy.Scope, string, ProviderSupplierInput, string) (channel.Supplier, error)
	CreateSupplierContract(context.Context, tenancy.Scope, SupplierContractInput, string) (channel.SupplierContract, error)
	TransitionSupplierContract(context.Context, tenancy.Scope, string, string, string) (channel.SupplierContract, error)
	ReviewSupplierContract(context.Context, tenancy.Scope, string, ReviewInput, string) (channel.SupplierContract, error)
	CreateSupplierRate(context.Context, tenancy.Scope, SupplierRateInput, string) (channel.SupplierRate, error)
	RecordSupplierQuality(context.Context, tenancy.Scope, SupplierQualityInput, string) (channel.SupplierQualityRecord, error)
	CreateDeveloper(context.Context, tenancy.Scope, DeveloperInput, string) (marketplace.Developer, error)
	CreatePublisher(context.Context, tenancy.Scope, PublisherInput, string) (marketplace.Publisher, error)
	CreateListing(context.Context, tenancy.Scope, ListingInput, string) (marketplace.Listing, error)
	CreateListingVersion(context.Context, tenancy.Scope, string, ListingVersionInput, string) (marketplace.ListingVersion, error)
	TransitionListing(context.Context, tenancy.Scope, string, string, string) (marketplace.Listing, error)
	ReviewListing(context.Context, tenancy.Scope, string, ListingReviewInput, string) (marketplace.Review, error)
	RunSandbox(context.Context, tenancy.Scope, SandboxRunInput, string) (marketplace.SandboxRun, error)
	RecordListingQuality(context.Context, tenancy.Scope, ListingQualityInput, string) (marketplace.QualityScore, error)
	CreateMarketplaceDispute(context.Context, tenancy.Scope, MarketplaceDisputeInput, string) (marketplace.Dispute, error)
	ResolveMarketplaceDispute(context.Context, tenancy.Scope, string, DisputeResolutionInput, string) (marketplace.Dispute, error)
	RequestTakedown(context.Context, tenancy.Scope, TakedownInput, string) (marketplace.Takedown, error)
	ReviewTakedown(context.Context, tenancy.Scope, string, ReviewInput, string) (marketplace.Takedown, error)
}
