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
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	BlueprintID string    `json:"blueprint_id"`
	Version     int       `json:"version"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type ProductVersion struct {
	ID                  string              `json:"id"`
	TenantID            string              `json:"tenant_id"`
	ProductID           string              `json:"product_id"`
	Version             int                 `json:"version"`
	InputSchema         schema.Definition   `json:"input_schema"`
	OutputSchema        schema.Definition   `json:"output_schema"`
	FormSchema          schema.Definition   `json:"form_schema"`
	CapabilityIDs       []string            `json:"capability_ids"`
	Workflow            workflow.Definition `json:"workflow"`
	MeteringID          string              `json:"metering_id"`
	PriceBookID         string              `json:"price_book_id"`
	RoutePolicyID       string              `json:"route_policy_id"`
	DeliveryMode        string              `json:"delivery_mode"`
	ComplianceProfileID string              `json:"compliance_profile_id"`
	ComplianceProfile   map[string]any      `json:"compliance_profile"`
	GrowthPlaybook      map[string]any      `json:"growth_playbook"`
	CreatedBy           string              `json:"created_by"`
	CreatedAt           time.Time           `json:"created_at"`
}
type SKU struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ProductID string    `json:"product_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}
type SKUVersion struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	SKUID             string         `json:"sku_id"`
	ProductVersionID  string         `json:"product_version_id"`
	WorkflowVersionID string         `json:"workflow_version_id"`
	MeteringVersionID string         `json:"metering_version_id"`
	PricingVersionID  string         `json:"pricing_version_id"`
	RoutingVersionID  string         `json:"routing_version_id"`
	Version           int            `json:"version"`
	Entitlements      map[string]any `json:"entitlements"`
	CreatedBy         string         `json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
}
type Publication struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ProductID        string    `json:"product_id"`
	ProductVersionID string    `json:"product_version_id"`
	Status           string    `json:"status"`
	PublishedBy      string    `json:"published_by"`
	PublishedAt      time.Time `json:"published_at"`
}

type SKUDetail struct {
	SKU
	Versions []SKUVersion `json:"versions"`
}

type ProductDetail struct {
	Product
	Versions     []ProductVersion `json:"versions"`
	SKUs         []SKUDetail      `json:"skus"`
	Publications []Publication    `json:"publications"`
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
	formSchema := v.FormSchema
	if len(formSchema) == 0 {
		formSchema = v.InputSchema
	}
	if err := schema.Validate(formSchema); err != nil {
		return fmt.Errorf("form schema: %w", err)
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
	if v.MeteringID == "" || v.PriceBookID == "" || v.RoutePolicyID == "" || v.DeliveryMode == "" || (v.ComplianceProfileID == "" && len(v.ComplianceProfile) == 0) {
		return fmt.Errorf("metering, pricing, routing, delivery, and compliance bindings are required")
	}
	return nil
}
func Publish(product *Product, version ProductVersion, providers map[string]bool) (Publication, error) {
	if err := version.ValidateForPublication(providers); err != nil {
		return Publication{}, err
	}
	if product.Status == "published" {
		product.Version++
		product.UpdatedAt = time.Now().UTC()
		return Publication{ID: platform.NewID("pub"), TenantID: product.TenantID, ProductID: product.ID, ProductVersionID: version.ID, Status: "published", PublishedAt: time.Now().UTC()}, nil
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
	return Publication{ID: platform.NewID("pub"), TenantID: product.TenantID, ProductID: product.ID, ProductVersionID: version.ID, Status: "published", PublishedAt: time.Now().UTC()}, nil
}
