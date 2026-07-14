package application

import (
	"context"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
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
