package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func TestTransactionAndExecutionPersistence(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Transaction")
	otherTenantID := createSecurityTenant(t, pool, "Transaction Other")
	defer cleanupTransactionTenant(t, pool, tenantID)
	defer cleanupTransactionTenant(t, pool, otherTenantID)

	var opportunityID string
	if err := pool.QueryRow(ctx, `INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'Test Opportunity','incubating','test') RETURNING id`, tenantID).Scan(&opportunityID); err != nil {
		t.Fatal(err)
	}
	var blueprintID string
	if err := pool.QueryRow(ctx, `INSERT INTO business_blueprints(tenant_id,source_opportunity_id,name,version,status,definition,created_by,approved_by) VALUES($1,$2,'Test Business Blueprint',1,'approved','{}','test','test') RETURNING id`, tenantID, opportunityID).Scan(&blueprintID); err != nil {
		t.Fatal(err)
	}

	store := NewStore(pool)
	scope := tenancy.Scope{TenantID: tenantID, ActorID: "transaction-test", TraceID: "trace-transaction-test"}
	capabilityItem, err := store.CreateCapability(ctx, scope, "Test Capability", "Neutral execution capability", map[string]any{"fixture": true}, "transaction-capability")
	mustStore(t, err)
	provider, err := store.CreateProvider(ctx, scope, "Test Provider", "transaction-provider")
	mustStore(t, err)
	endpoint, err := store.CreateProviderEndpoint(ctx, scope, provider.ID, capabilityItem.ID, "mock_realtime", "v1", "transaction-endpoint")
	mustStore(t, err)
	product, err := store.CreateProduct(ctx, scope, blueprintID, "Test Product", "transaction-product")
	mustStore(t, err)
	productVersion, err := store.CreateProductVersion(ctx, scope, product.ID, neutralProductVersionInput(capabilityItem.ID), "transaction-product-version")
	mustStore(t, err)
	sku, err := store.CreateSKU(ctx, scope, product.ID, "TEST-SKU", "Test SKU", "transaction-sku")
	mustStore(t, err)
	skuVersion, err := store.CreateSKUVersion(ctx, scope, sku.ID, application.SKUVersionInput{ProductVersionID: productVersion.ID, Entitlements: map[string]any{"test_limit": 10}}, "transaction-sku-version")
	mustStore(t, err)
	_, err = store.PublishProduct(ctx, scope, product.ID, productVersion.ID, "transaction-publication")
	mustStore(t, err)

	quoteInput := application.QuoteInput{
		DealID: "Test Deal", CustomerID: "Test Customer", Currency: "USD", ValidUntil: time.Now().UTC().Add(24 * time.Hour),
		Items: []application.QuoteItemInput{{SKUVersionID: skuVersion.ID, Quantity: 2, Input: map[string]any{"input": "Test Order Input"}}},
	}
	quote, err := store.CreateQuote(ctx, scope, quoteInput, "transaction-quote")
	mustStore(t, err)
	if len(quote.Versions) != 1 || len(quote.Versions[0].Items) != 1 || quote.Versions[0].AmountMinor != 700 {
		t.Fatalf("unexpected server-calculated quote: %#v", quote)
	}
	replayed, err := store.CreateQuote(ctx, scope, quoteInput, "transaction-quote")
	mustStore(t, err)
	if replayed.ID != quote.ID {
		t.Fatalf("idempotent quote replay created a second aggregate: %s != %s", replayed.ID, quote.ID)
	}
	quote, err = store.TransitionQuote(ctx, scope, quote.ID, "accepted", "transaction-quote-accept")
	mustStore(t, err)
	if quote.Status != "accepted" {
		t.Fatalf("quote status=%s", quote.Status)
	}

	createdOrder, err := store.CreateOrder(ctx, scope, quote.Versions[0].ID, "transaction-order")
	mustStore(t, err)
	if createdOrder.Status != "created" || len(createdOrder.Items) != 1 || createdOrder.Items[0].Bindings.SKUVersionID != skuVersion.ID || createdOrder.AmountMinor != 700 {
		t.Fatalf("unexpected order version snapshot: %#v", createdOrder)
	}
	if _, err = store.GetOrder(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}, createdOrder.ID); err == nil {
		t.Fatal("cross-tenant order read succeeded")
	}
	if _, err = store.CreateQuote(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}, quoteInput, "cross-tenant-quote"); err == nil {
		t.Fatal("cross-tenant quote referenced another tenant's SKU version")
	}

	for _, next := range []string{"awaiting_payment", "paid", "provisioning"} {
		createdOrder, err = store.TransitionOrder(ctx, scope, createdOrder.ID, next, "transaction-order-"+next)
		mustStore(t, err)
		if createdOrder.Status != next {
			t.Fatalf("order status=%s", createdOrder.Status)
		}
	}
	if len(createdOrder.Subscriptions) != 1 || len(createdOrder.Entitlements) != 1 || len(createdOrder.Executions) != 1 || len(createdOrder.Deliveries) != 1 {
		t.Fatalf("fulfillment records were not created atomically: %#v", createdOrder)
	}
	execution := createdOrder.Executions[0]
	for _, transition := range []application.ExecutionTransitionInput{
		{To: "validating"},
		{To: "reserved", ProviderEndpointID: endpoint.ID},
		{To: "queued"},
		{To: "submitted", ExternalID: "test-external-execution"},
		{To: "processing"},
		{To: "succeeded", Output: map[string]any{"result": "Test Result", "units": 2}},
	} {
		execution, err = store.TransitionExecution(ctx, scope, execution.ID, transition, "transaction-execution-"+transition.To)
		mustStore(t, err)
	}
	usage, err := store.RecordUsage(ctx, scope, execution.ID, 2, time.Now().UTC(), "transaction-usage")
	mustStore(t, err)
	if usage.Quantity != 2 || usage.MeterID != productVersion.MeteringID {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	cost, err := store.RecordProviderCost(ctx, scope, execution.ID, endpoint.ID, "USD", 250, "transaction-cost")
	mustStore(t, err)
	if cost.AmountMinor != 250 {
		t.Fatalf("unexpected provider cost: %#v", cost)
	}
	charge, err := store.CreateCustomerCharge(ctx, scope, execution.ID, "transaction-charge")
	mustStore(t, err)
	if charge.AmountMinor != 700 || charge.Status != "calculated" {
		t.Fatalf("unexpected customer charge: %#v", charge)
	}

	delivery := createdOrder.Deliveries[0]
	delivery, err = store.TransitionDelivery(ctx, scope, delivery.ID, "in_progress", "transaction-delivery-start")
	mustStore(t, err)
	delivery, err = store.TransitionDelivery(ctx, scope, delivery.ID, "completed", "transaction-delivery-complete")
	mustStore(t, err)
	if delivery.Status != "completed" {
		t.Fatalf("delivery status=%s", delivery.Status)
	}
	createdOrder, err = store.TransitionOrder(ctx, scope, createdOrder.ID, "active", "transaction-order-active")
	mustStore(t, err)
	if createdOrder.Subscriptions[0].Status != "active" || createdOrder.Entitlements[0].Status != "active" {
		t.Fatalf("activation did not enable subscription and entitlements: %#v", createdOrder)
	}
	execution, err = store.TransitionExecution(ctx, scope, execution.ID, application.ExecutionTransitionInput{To: "reconciling"}, "transaction-execution-reconciling")
	mustStore(t, err)
	_, err = store.TransitionExecution(ctx, scope, execution.ID, application.ExecutionTransitionInput{To: "settled"}, "transaction-execution-settled")
	mustStore(t, err)
	createdOrder, err = store.TransitionOrder(ctx, scope, createdOrder.ID, "completed", "transaction-order-completed")
	mustStore(t, err)
	if createdOrder.Status != "completed" || len(createdOrder.Usage) != 1 || len(createdOrder.ProviderCosts) != 1 || len(createdOrder.CustomerCharges) != 1 {
		t.Fatalf("incomplete transaction facts: %#v", createdOrder)
	}

	var audits, events int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND action IN ('quote.create','quote.transition','order.create','order.transition','execution.transition','delivery.transition','usage.record','provider_cost.record','customer_charge.create')`, tenantID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_type IN ('quote','order','execution_order','delivery_project')`, tenantID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if audits < 17 || events < 17 {
		t.Fatalf("transactional audit/outbox chain incomplete: audits=%d events=%d", audits, events)
	}
}

func cleanupTransactionTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{
		"customer_charges", "provider_costs", "usage_records", "delivery_projects", "execution_orders",
		"entitlements", "subscriptions", "order_items", "orders", "quote_version_items", "quote_versions", "quotes",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	cleanupProductFactoryTenant(t, pool, tenantID)
}
