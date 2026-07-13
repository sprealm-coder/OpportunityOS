package catalog

import (
	"fmt"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

type Product struct {
	ID, TenantID, Name, Status, BlueprintID string
	Version                                 int
	CreatedAt, UpdatedAt                    time.Time
}
type ProductVersion struct {
	ID, TenantID, ProductID                                                   string
	Version                                                                   int
	InputSchema, OutputSchema                                                 schema.Definition
	CapabilityIDs                                                             []string
	Workflow                                                                  workflow.Definition
	MeteringID, PriceBookID, RoutePolicyID, DeliveryMode, ComplianceProfileID string
}
type SKU struct{ ID, TenantID, ProductID, Code string }
type SKUVersion struct {
	ID, TenantID, SKUID, ProductVersionID, WorkflowVersionID, MeteringVersionID, PricingVersionID, RoutingVersionID string
	Version                                                                                                         int
}
type Publication struct {
	ID, TenantID, ProductVersionID, Status string
	PublishedAt                            time.Time
}

func Draft(tenantID, blueprintID, name string) (Product, ProductVersion) {
	now := time.Now().UTC()
	product := Product{ID: platform.NewID("prod"), TenantID: tenantID, Name: name, Status: "draft", BlueprintID: blueprintID, Version: 1, CreatedAt: now, UpdatedAt: now}
	version := ProductVersion{ID: platform.NewID("prodv"), TenantID: tenantID, ProductID: product.ID, Version: 1}
	return product, version
}

func (p *Product) Transition(to string) error {
	if err := state.Product.Transition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	p.Version++
	p.UpdatedAt = time.Now().UTC()
	return nil
}
func (v ProductVersion) ValidateForPublication(providerCapabilityIDs map[string]bool) error {
	if err := schema.Validate(v.InputSchema); err != nil {
		return fmt.Errorf("input schema: %w", err)
	}
	if err := schema.Validate(v.OutputSchema); err != nil {
		return fmt.Errorf("output schema: %w", err)
	}
	if len(v.CapabilityIDs) == 0 {
		return fmt.Errorf("at least one capability is required")
	}
	for _, id := range v.CapabilityIDs {
		if !providerCapabilityIDs[id] {
			return fmt.Errorf("capability %s has no provider", id)
		}
	}
	if err := v.Workflow.Validate(); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if v.MeteringID == "" || v.PriceBookID == "" || v.RoutePolicyID == "" || v.DeliveryMode == "" || v.ComplianceProfileID == "" {
		return fmt.Errorf("metering, pricing, routing, delivery, and compliance bindings are required")
	}
	return nil
}
func Publish(product *Product, version ProductVersion, providers map[string]bool) (Publication, error) {
	if err := version.ValidateForPublication(providers); err != nil {
		return Publication{}, err
	}
	if product.Status == "draft" {
		if err := product.Transition("validating"); err != nil {
			return Publication{}, err
		}
	}
	if product.Status == "validating" {
		if err := product.Transition("ready"); err != nil {
			return Publication{}, err
		}
	}
	if err := product.Transition("published"); err != nil {
		return Publication{}, err
	}
	return Publication{ID: platform.NewID("pub"), TenantID: product.TenantID, ProductVersionID: version.ID, Status: "published", PublishedAt: time.Now().UTC()}, nil
}
