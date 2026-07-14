package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/pricing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/routing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

func TestProductFactoryPersistsAndPublishesImmutableVersion(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Product Factory")
	otherTenantID := createSecurityTenant(t, pool, "Product Factory Other")
	defer cleanupProductFactoryTenant(t, pool, tenantID)
	defer cleanupProductFactoryTenant(t, pool, otherTenantID)
	var opportunityID string
	if err := pool.QueryRow(ctx, `INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'Test Opportunity','incubating','test') RETURNING id`, tenantID).Scan(&opportunityID); err != nil {
		t.Fatal(err)
	}
	var blueprintID string
	if err := pool.QueryRow(ctx, `INSERT INTO business_blueprints(tenant_id,source_opportunity_id,name,version,status,definition,created_by,approved_by) VALUES($1,$2,'Test Business Blueprint',1,'approved','{}','test','test') RETURNING id`, tenantID, opportunityID).Scan(&blueprintID); err != nil {
		t.Fatal(err)
	}

	store := NewStore(pool)
	scope := tenancy.Scope{TenantID: tenantID, ActorID: "product-test", TraceID: "trace-product-test"}
	capabilityItem, err := store.CreateCapability(ctx, scope, "Test Capability", "Neutral execution capability", map[string]any{"fixture": true}, "capability")
	mustStore(t, err)
	provider, err := store.CreateProvider(ctx, scope, "Test Provider", "provider")
	mustStore(t, err)
	_, err = store.CreateProviderEndpoint(ctx, scope, provider.ID, capabilityItem.ID, "mock_realtime", "v1", "endpoint")
	mustStore(t, err)
	product, err := store.CreateProduct(ctx, scope, blueprintID, "Test Product", "product")
	mustStore(t, err)

	versionInput := neutralProductVersionInput(capabilityItem.ID)
	version, err := store.CreateProductVersion(ctx, scope, product.ID, versionInput, "version-1")
	mustStore(t, err)
	if version.Version != 1 || version.Workflow.ID == "" || version.PriceBookID == "" {
		t.Fatalf("unexpected product version: %#v", version)
	}
	if _, err = store.PublishProduct(ctx, scope, product.ID, version.ID, "publish-without-sku"); err == nil {
		t.Fatal("product published without a SKU version")
	}
	sku, err := store.CreateSKU(ctx, scope, product.ID, "TEST-SKU", "Test SKU", "sku")
	mustStore(t, err)
	skuVersion, err := store.CreateSKUVersion(ctx, scope, sku.ID, application.SKUVersionInput{ProductVersionID: version.ID, Entitlements: map[string]any{"test_limit": 10}}, "sku-version")
	mustStore(t, err)
	if skuVersion.WorkflowVersionID != version.Workflow.ID || skuVersion.PricingVersionID != version.PriceBookID {
		t.Fatalf("SKU version bindings do not match product version: %#v", skuVersion)
	}
	publication, err := store.PublishProduct(ctx, scope, product.ID, version.ID, "publish")
	mustStore(t, err)
	if publication.Status != "published" || publication.ProductVersionID != version.ID {
		t.Fatalf("unexpected publication: %#v", publication)
	}

	detail, err := store.GetProduct(ctx, scope, product.ID)
	mustStore(t, err)
	if detail.Status != "published" || len(detail.Versions) != 1 || len(detail.SKUs) != 1 || len(detail.SKUs[0].Versions) != 1 || len(detail.Publications) != 1 {
		t.Fatalf("incomplete product detail: %#v", detail)
	}
	secondVersion, err := store.CreateProductVersion(ctx, scope, product.ID, versionInput, "version-2")
	mustStore(t, err)
	if secondVersion.Version != 2 || secondVersion.ID == version.ID {
		t.Fatalf("immutable version sequence failed: %#v", secondVersion)
	}
	if _, err = store.GetProduct(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}, product.ID); err == nil {
		t.Fatal("cross-tenant product read succeeded")
	}

	var audits, events int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND object_type IN ('capability','provider','provider_endpoint','product','product_version','sku','sku_version','publication')`, tenantID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1`, tenantID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if audits < 9 || events < 9 {
		t.Fatalf("expected transactional audit/outbox chain, audit=%d events=%d", audits, events)
	}
}

func neutralProductVersionInput(capabilityID string) application.ProductVersionInput {
	inputSchema := schema.Definition{"type": "object", "properties": map[string]any{"input": map[string]any{"type": "string"}}, "required": []any{"input"}}
	outputSchema := schema.Definition{"type": "object", "properties": map[string]any{"result": map[string]any{"type": "string"}, "units": map[string]any{"type": "integer"}}}
	return application.ProductVersionInput{
		InputSchema: inputSchema, OutputSchema: outputSchema, FormSchema: inputSchema,
		CapabilityIDs: []string{capabilityID},
		Workflow: workflow.Definition{Name: "Test Workflow", Version: 1,
			Nodes: []workflow.Node{{ID: "start", Type: workflow.Start}, {ID: "validate", Type: workflow.Validate}, {ID: "execute", Type: workflow.RealtimeCall}, {ID: "meter", Type: workflow.Meter}, {ID: "end", Type: workflow.End}},
			Edges: []workflow.Edge{{From: "start", To: "validate"}, {From: "validate", To: "execute"}, {From: "execute", To: "meter"}, {From: "meter", To: "end"}}},
		Metering:     pricing.MeteringDefinition{Name: "Test Meter", Unit: "test_unit", Field: "units", Version: 1},
		PriceBook:    pricing.PriceBook{Currency: "USD", Version: 1, Rules: []pricing.Rule{{Kind: "flat", FlatMinor: 500}, {Kind: "per_unit", UnitMinor: 100}}},
		RoutePolicy:  routing.Policy{Name: "Test Route", Strategy: "priority", Version: 1},
		DeliveryMode: "workflow", ComplianceProfile: map[string]any{"classification": "test"},
		GrowthPlaybook: map[string]any{"fixture": true},
	}
}

func cleanupProductFactoryTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"publications", "product_compliance_bindings", "product_growth_bindings", "product_output_definitions",
		"product_form_definitions", "product_routing_bindings", "product_pricing_bindings", "product_metering_bindings",
		"product_workflow_bindings", "product_capability_bindings", "sku_versions", "skus", "product_versions",
		"price_rules", "workflow_definitions", "metering_definitions", "price_books", "route_policies", "products",
		"provider_endpoints", "providers", "capabilities", "command_idempotency", "audit_log", "outbox_events",
		"business_blueprints", "opportunities", "memberships", "brands",
	}
	for _, table := range tables {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, tenantID); err != nil {
		t.Errorf("cleanup tenant %s: %v", tenantID, err)
	}
}
