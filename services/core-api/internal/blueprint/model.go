package blueprint

import (
	"fmt"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type BusinessBlueprint struct {
	ID                     string           `json:"id"`
	TenantID               string           `json:"tenant_id"`
	Name                   string           `json:"name"`
	Description            string           `json:"description"`
	Version                int              `json:"version"`
	Status                 string           `json:"status"`
	SourceOpportunityID    string           `json:"source_opportunity_id"`
	BusinessModel          map[string]any   `json:"business_model"`
	TargetMarketDefinition map[string]any   `json:"target_market_definition"`
	ValueProposition       string           `json:"value_proposition"`
	RevenueModel           map[string]any   `json:"revenue_model"`
	DeliveryModel          map[string]any   `json:"delivery_model"`
	RequiredCapabilities   []string         `json:"required_capabilities"`
	ProductDefinitions     []map[string]any `json:"product_definitions"`
	WorkflowDefinitions    []map[string]any `json:"workflow_definitions"`
	MeteringDefinitions    []map[string]any `json:"metering_definitions"`
	PricingDefinitions     []map[string]any `json:"pricing_definitions"`
	GrowthPlaybook         map[string]any   `json:"growth_playbook"`
	ComplianceProfile      map[string]any   `json:"compliance_profile"`
	LaunchPlan             map[string]any   `json:"launch_plan"`
	ValidationMetrics      map[string]any   `json:"validation_metrics"`
	CreatedBy              string           `json:"created_by"`
	ApprovedBy             string           `json:"approved_by,omitempty"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
}

func New(tenantID, actorID, opportunityID, name, description string) BusinessBlueprint {
	now := time.Now().UTC()
	return BusinessBlueprint{ID: platform.NewID("bp"), TenantID: tenantID, Name: name, Description: description, Version: 1, Status: "draft", SourceOpportunityID: opportunityID, BusinessModel: map[string]any{}, TargetMarketDefinition: map[string]any{}, RevenueModel: map[string]any{}, DeliveryModel: map[string]any{}, GrowthPlaybook: map[string]any{}, ComplianceProfile: map[string]any{}, LaunchPlan: map[string]any{}, ValidationMetrics: map[string]any{}, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
}

func (b *BusinessBlueprint) Transition(to, actorID string) error {
	if err := state.Blueprint.Transition(b.Status, to); err != nil {
		return err
	}
	b.Status = to
	b.Version++
	b.UpdatedAt = time.Now().UTC()
	if to == "approved" {
		b.ApprovedBy = actorID
	}
	return nil
}

func (b BusinessBlueprint) Clone(actorID string) BusinessBlueprint {
	clone := b
	clone.ID = platform.NewID("bp")
	clone.Status = "draft"
	clone.Version = 1
	clone.ApprovedBy = ""
	clone.CreatedBy = actorID
	clone.CreatedAt = time.Now().UTC()
	clone.UpdatedAt = clone.CreatedAt
	return clone
}
func (b BusinessBlueprint) NextVersion(actorID string) BusinessBlueprint {
	next := b
	next.Version++
	next.Status = "draft"
	next.ApprovedBy = ""
	next.CreatedBy = actorID
	next.UpdatedAt = time.Now().UTC()
	return next
}

func (b BusinessBlueprint) ValidateCompleteness() error {
	missing := []string{}
	if b.Name == "" {
		missing = append(missing, "name")
	}
	if b.SourceOpportunityID == "" {
		missing = append(missing, "source_opportunity_id")
	}
	if b.ValueProposition == "" {
		missing = append(missing, "value_proposition")
	}
	if len(b.RequiredCapabilities) == 0 {
		missing = append(missing, "required_capabilities")
	}
	if len(b.ProductDefinitions) == 0 {
		missing = append(missing, "product_definitions")
	}
	if len(b.WorkflowDefinitions) == 0 {
		missing = append(missing, "workflow_definitions")
	}
	if len(b.MeteringDefinitions) == 0 {
		missing = append(missing, "metering_definitions")
	}
	if len(b.PricingDefinitions) == 0 {
		missing = append(missing, "pricing_definitions")
	}
	if len(b.ComplianceProfile) == 0 {
		missing = append(missing, "compliance_profile")
	}
	if len(missing) > 0 {
		return fmt.Errorf("blueprint incomplete: %v", missing)
	}
	return nil
}
