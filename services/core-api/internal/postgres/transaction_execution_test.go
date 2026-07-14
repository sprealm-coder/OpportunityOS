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

	createdOrder, err = store.TransitionOrder(ctx, scope, createdOrder.ID, "awaiting_payment", "transaction-order-awaiting-payment")
	mustStore(t, err)
	if _, err = store.TransitionOrder(ctx, scope, createdOrder.ID, "paid", "transaction-order-paid-without-hold"); err == nil {
		t.Fatal("order payment succeeded without held funds")
	}
	wallet, err := store.CreateWallet(ctx, scope, application.WalletInput{OwnerType: "customer", OwnerID: "Test Customer", Currency: "USD"}, "transaction-wallet")
	mustStore(t, err)
	adjustment, err := store.PostWalletAdjustment(ctx, scope, wallet.ID, application.WalletAdjustmentInput{Direction: "credit", AmountMinor: 1200, Reason: "Test funding"}, "transaction-wallet-funding")
	mustStore(t, err)
	replayedAdjustment, err := store.PostWalletAdjustment(ctx, scope, wallet.ID, application.WalletAdjustmentInput{Direction: "credit", AmountMinor: 1200, Reason: "Test funding"}, "transaction-wallet-funding")
	mustStore(t, err)
	if replayedAdjustment.ID != adjustment.ID {
		t.Fatal("wallet adjustment idempotency replay created a second record")
	}
	hold, err := store.PlaceOrderHold(ctx, scope, createdOrder.ID, application.HoldInput{WalletID: wallet.ID, AmountMinor: 800}, "transaction-hold")
	mustStore(t, err)
	if hold.RemainingMinor != 800 || hold.Status != "active" {
		t.Fatalf("unexpected hold: %#v", hold)
	}
	for _, next := range []string{"paid", "provisioning"} {
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
	chargeTransaction, err := store.PostCustomerCharge(ctx, scope, charge.ID, application.ChargePostingInput{HoldID: hold.ID}, "transaction-charge-post")
	mustStore(t, err)
	if len(chargeTransaction.Entries) != 2 {
		t.Fatalf("charge transaction entries=%d", len(chargeTransaction.Entries))
	}
	hold, err = store.ReleaseHold(ctx, scope, hold.ID, application.ReleaseInput{}, "transaction-hold-release")
	mustStore(t, err)
	if hold.Status != "captured" || hold.RemainingMinor != 0 || hold.CapturedMinor != 700 || hold.ReleasedMinor != 100 {
		t.Fatalf("unexpected captured hold: %#v", hold)
	}
	discrepancyRun, err := store.RunReconciliation(ctx, scope, application.ReconciliationInput{OrderID: createdOrder.ID}, "transaction-reconciliation-missing-payable")
	mustStore(t, err)
	if discrepancyRun.Status != "discrepancy" || discrepancyRun.DiscrepancyCount != 1 {
		t.Fatalf("expected missing-payable discrepancy: %#v", discrepancyRun)
	}
	payable, err := store.CreateProviderPayable(ctx, scope, cost.ID, "transaction-provider-payable")
	mustStore(t, err)
	commission, err := store.CreateCommission(ctx, scope, charge.ID, application.CommissionInput{BeneficiaryType: "reseller", BeneficiaryID: "Test Reseller", AmountMinor: 70}, "transaction-commission")
	mustStore(t, err)
	reconciliation, err := store.RunReconciliation(ctx, scope, application.ReconciliationInput{OrderID: createdOrder.ID}, "transaction-reconciliation")
	mustStore(t, err)
	if reconciliation.Status != "matched" || reconciliation.CheckedCount != 3 || reconciliation.DiscrepancyCount != 0 {
		t.Fatalf("unexpected reconciliation: %#v", reconciliation)
	}
	_, err = store.CreateSettlement(ctx, scope, application.SettlementInput{SourceType: "provider_payable", SourceID: payable.ID}, "transaction-provider-settlement")
	mustStore(t, err)
	_, err = store.CreateSettlement(ctx, scope, application.SettlementInput{SourceType: "commission", SourceID: commission.ID}, "transaction-commission-settlement")
	mustStore(t, err)

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
	refund, err := store.RefundCustomerCharge(ctx, scope, charge.ID, application.RefundInput{WalletID: wallet.ID, AmountMinor: 100, Reason: "Test partial refund"}, "transaction-refund")
	mustStore(t, err)
	if refund.AmountMinor != 100 || refund.Status != "posted" {
		t.Fatalf("unexpected refund: %#v", refund)
	}
	if _, err = store.RefundCustomerCharge(ctx, scope, charge.ID, application.RefundInput{WalletID: wallet.ID, AmountMinor: 700, Reason: "Test excessive refund"}, "transaction-refund-excessive"); err == nil {
		t.Fatal("refund exceeded the unrefunded customer charge amount")
	}
	financeOverview, err := store.ListFinance(ctx, scope)
	mustStore(t, err)
	if len(financeOverview.Wallets) != 1 || financeOverview.Wallets[0].AvailableMinor != 600 || financeOverview.Wallets[0].HeldMinor != 0 {
		t.Fatalf("unexpected wallet balances: %#v", financeOverview.Wallets)
	}
	if len(financeOverview.Transactions) != 9 || len(financeOverview.ProviderPayables) != 1 || financeOverview.ProviderPayables[0].Status != "settled" || len(financeOverview.Commissions) != 1 || financeOverview.Commissions[0].Status != "settled" {
		t.Fatalf("incomplete finance overview: %#v", financeOverview)
	}
	if otherFinance, otherErr := store.ListFinance(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}); otherErr != nil || len(otherFinance.Wallets) != 0 || len(otherFinance.Transactions) != 0 {
		t.Fatalf("cross-tenant finance visibility: overview=%#v err=%v", otherFinance, otherErr)
	}
	for _, transaction := range financeOverview.Transactions {
		var debitMinor, creditMinor int64
		for _, entry := range transaction.Entries {
			if entry.Direction == "debit" {
				debitMinor += entry.AmountMinor
			} else {
				creditMinor += entry.AmountMinor
			}
		}
		if debitMinor != creditMinor || debitMinor == 0 {
			t.Fatalf("unbalanced persisted transaction %s: debit=%d credit=%d", transaction.ID, debitMinor, creditMinor)
		}
	}
	immutableTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = immutableTx.Exec(ctx, `SELECT set_config('app.tenant_id',$1,true)`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err = immutableTx.Exec(ctx, `SET LOCAL ROLE opportunity_app`); err != nil {
		t.Fatal(err)
	}
	if _, err = immutableTx.Exec(ctx, `UPDATE ledger_entries SET amount_minor=amount_minor+1 WHERE tenant_id=$1 AND transaction_id=$2`, tenantID, chargeTransaction.ID); err == nil {
		t.Fatal("runtime role modified append-only ledger entry")
	}
	_ = immutableTx.Rollback(ctx)

	var audits, events, financeAudits, financeEvents int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND action IN ('quote.create','quote.transition','order.create','order.transition','execution.transition','delivery.transition','usage.record','provider_cost.record','customer_charge.create')`, tenantID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_type IN ('quote','order','execution_order','delivery_project')`, tenantID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if audits < 17 || events < 17 {
		t.Fatalf("transactional audit/outbox chain incomplete: audits=%d events=%d", audits, events)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND action IN ('wallet.create','wallet.adjust','hold.create','hold.release','customer_charge.post','customer_charge.refund','provider_payable.create','commission.create','settlement.create','reconciliation.run')`, tenantID).Scan(&financeAudits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_type IN ('wallet','hold','customer_charge','provider_payable','commission','settlement','reconciliation_run')`, tenantID).Scan(&financeEvents); err != nil {
		t.Fatal(err)
	}
	if financeAudits < 12 || financeEvents < 12 {
		t.Fatalf("finance audit/outbox chain incomplete: audits=%d events=%d", financeAudits, financeEvents)
	}
}

func cleanupTransactionTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{
		"reconciliation_discrepancies", "reconciliation_items", "reconciliation_runs",
		"settlements", "settlement_reserves", "chargeback_reserves", "promotional_credits",
		"refunds", "commissions", "provider_payables", "hold_releases", "holds",
		"financial_adjustments", "payment_receivables", "balance_snapshots", "ledger_entries",
		"customer_charges", "provider_costs", "usage_records", "delivery_projects", "execution_orders",
		"entitlements", "subscriptions", "order_items", "orders", "quote_version_items", "quote_versions", "quotes",
		"ledger_transactions", "ledger_accounts", "wallets",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	cleanupProductFactoryTenant(t, pool, tenantID)
}
