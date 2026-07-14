package application

import (
	"context"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
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

type Store interface {
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
}
