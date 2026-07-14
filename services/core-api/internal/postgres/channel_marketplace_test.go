package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func TestChannelAndMarketplacePersistence(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Channel")
	otherTenantID := createSecurityTenant(t, pool, "Channel Other")
	defer cleanupChannelTenant(t, pool, tenantID)
	defer cleanupChannelTenant(t, pool, otherTenantID)

	store := NewStore(pool)
	creator := tenancy.Scope{TenantID: tenantID, ActorID: "channel-operator", TraceID: "trace-channel"}
	reviewer := tenancy.Scope{TenantID: tenantID, ActorID: "channel-reviewer", TraceID: "trace-channel-review"}

	segment, err := store.CreateMarketSegment(ctx, creator, application.MarketSegmentInput{Name: "Test Market Segment", Definition: map[string]any{"fixture": true}}, "channel-segment")
	mustStore(t, err)
	lead, err := store.CreateLead(ctx, creator, application.LeadInput{MarketSegmentID: segment.ID, Name: "Test Customer"}, "channel-lead")
	mustStore(t, err)
	capabilityItem, err := store.CreateCapability(ctx, creator, "Test Capability", "Neutral channel fixture", map[string]any{"fixture": true}, "channel-capability")
	mustStore(t, err)
	provider, err := store.CreateProvider(ctx, creator, "Test Provider", "channel-provider")
	mustStore(t, err)
	endpoint, err := store.CreateProviderEndpoint(ctx, creator, provider.ID, capabilityItem.ID, "mock_realtime", "v1", "channel-endpoint")
	mustStore(t, err)

	level, err := store.CreateResellerLevel(ctx, creator, application.ResellerLevelInput{Name: "Test Reseller Level", Rank: 1, DefaultCommissionBPS: 1000}, "channel-level")
	mustStore(t, err)
	resellerA, err := store.CreateReseller(ctx, creator, application.ResellerInput{LevelID: level.ID, Name: "Test Reseller"}, "channel-reseller-a")
	mustStore(t, err)
	resellerB, err := store.CreateReseller(ctx, creator, application.ResellerInput{LevelID: level.ID, Name: "Test Reseller Two"}, "channel-reseller-b")
	mustStore(t, err)
	rule, err := store.CreateAttributionRule(ctx, creator, application.AttributionRuleInput{Name: "Test Attribution Rule", Priority: 1, Definition: map[string]any{"method": "first_verified_evidence"}}, "channel-attribution-rule")
	mustStore(t, err)
	ownership, err := store.AssignLeadOwnership(ctx, creator, application.LeadOwnershipInput{LeadID: lead.ID, ResellerID: resellerA.ID, AttributionRuleID: rule.ID, ProtectionDays: 30}, "channel-lead-ownership")
	mustStore(t, err)
	replayedOwnership, err := store.AssignLeadOwnership(ctx, creator, application.LeadOwnershipInput{LeadID: lead.ID, ResellerID: resellerA.ID, AttributionRuleID: rule.ID, ProtectionDays: 30}, "channel-lead-ownership")
	mustStore(t, err)
	if replayedOwnership.ID != ownership.ID {
		t.Fatal("lead ownership idempotency replay created a second record")
	}

	transfer, err := store.CreateTransferRequest(ctx, creator, application.TransferRequestInput{OwnershipType: "lead", OwnershipID: ownership.ID, ToResellerID: resellerB.ID, Rationale: "Test verified transfer"}, "channel-transfer")
	mustStore(t, err)
	if _, err = store.ReviewTransfer(ctx, creator, transfer.ID, application.ReviewInput{Decision: "approved", Rationale: "self review"}, "channel-transfer-self-review"); err == nil {
		t.Fatal("ownership transfer requester approved their own request")
	}
	transfer, err = store.ReviewTransfer(ctx, reviewer, transfer.ID, application.ReviewInput{Decision: "approved", Rationale: "Test independent approval"}, "channel-transfer-review")
	mustStore(t, err)
	if transfer.Status != "approved" {
		t.Fatalf("transfer status=%s", transfer.Status)
	}
	customerOwnership, err := store.CreateCustomerOwnership(ctx, creator, application.CustomerOwnershipInput{CustomerID: "Test Customer", ResellerID: resellerB.ID, SourceLeadOwnershipID: ownership.ID, ProtectionDays: 90}, "channel-customer-ownership")
	mustStore(t, err)
	if customerOwnership.ResellerID != resellerB.ID {
		t.Fatal("customer ownership did not preserve transferred reseller")
	}

	commissionRule, err := store.CreateCommissionRule(ctx, creator, application.CommissionRuleInput{Name: "Test Commission Rule", ResellerID: resellerB.ID, BasisPoints: 1000}, "channel-commission-rule")
	mustStore(t, err)
	commissionID := seedCanonicalChannelFinance(t, pool, tenantID, resellerB.ID, provider.ID, endpoint.ID)
	commissionLock, err := store.LockCommission(ctx, creator, application.CommissionLockInput{CommissionID: commissionID, CommissionRuleID: commissionRule.ID, ResellerID: resellerB.ID}, "channel-commission-lock")
	mustStore(t, err)
	if commissionLock.CommissionID != commissionID || commissionLock.Status != "posted" {
		t.Fatalf("commission lock is not linked to canonical commission: %#v", commissionLock)
	}
	_, err = store.CreateSettlementCycle(ctx, creator, application.SettlementCycleInput{ResellerID: resellerB.ID, Name: "Test Settlement Cycle", PeriodStart: time.Now().UTC().AddDate(0, 0, -30), PeriodEnd: time.Now().UTC()}, "channel-cycle")
	mustStore(t, err)

	supplier, err := store.CreateSupplier(ctx, creator, application.SupplierInput{Name: "Test Supplier"}, "channel-supplier")
	mustStore(t, err)
	_, err = store.BindSupplierCapability(ctx, creator, application.SupplierCapabilityInput{SupplierID: supplier.ID, CapabilityID: capabilityItem.ID}, "channel-supplier-capability")
	mustStore(t, err)
	_, err = store.BindProviderSupplier(ctx, creator, provider.ID, application.ProviderSupplierInput{SupplierID: supplier.ID}, "channel-provider-supplier")
	mustStore(t, err)
	contract, err := store.CreateSupplierContract(ctx, creator, application.SupplierContractInput{SupplierID: supplier.ID, ProviderID: provider.ID, Name: "Test Supplier Contract", Currency: "USD", Terms: map[string]any{"settlement_days": 30}}, "channel-contract")
	mustStore(t, err)
	contract, err = store.TransitionSupplierContract(ctx, creator, contract.ID, "pending_approval", "channel-contract-submit")
	mustStore(t, err)
	if _, err = store.ReviewSupplierContract(ctx, creator, contract.ID, application.ReviewInput{Decision: "approved"}, "channel-contract-self-review"); err == nil {
		t.Fatal("supplier contract creator approved their own contract")
	}
	contract, err = store.ReviewSupplierContract(ctx, reviewer, contract.ID, application.ReviewInput{Decision: "approved", Rationale: "Test terms verified"}, "channel-contract-review")
	mustStore(t, err)
	_, err = store.CreateSupplierRate(ctx, creator, application.SupplierRateInput{ContractID: contract.ID, CapabilityID: capabilityItem.ID, Unit: "test_unit", RateMinor: 125}, "channel-rate")
	mustStore(t, err)
	contract, err = store.TransitionSupplierContract(ctx, creator, contract.ID, "active", "channel-contract-active")
	mustStore(t, err)
	if contract.Status != "active" {
		t.Fatalf("supplier contract status=%s", contract.Status)
	}
	_, err = store.RecordSupplierQuality(ctx, creator, application.SupplierQualityInput{SupplierID: supplier.ID, ProviderID: provider.ID, ProviderEndpointID: endpoint.ID, Metric: "test_success_rate", ScoreBPS: 9500, Evidence: map[string]any{"sample_size": 10}, PeriodStart: time.Now().UTC().AddDate(0, 0, -30), PeriodEnd: time.Now().UTC()}, "channel-quality")
	mustStore(t, err)

	developer, err := store.CreateDeveloper(ctx, creator, application.DeveloperInput{Name: "Test Developer"}, "channel-developer")
	mustStore(t, err)
	replayedDeveloper, err := store.CreateDeveloper(ctx, creator, application.DeveloperInput{Name: "Test Developer"}, "channel-developer")
	mustStore(t, err)
	if replayedDeveloper.ID != developer.ID {
		t.Fatal("developer idempotency replay created a second record")
	}
	publisher, err := store.CreatePublisher(ctx, creator, application.PublisherInput{DeveloperID: developer.ID, Name: "Test Publisher"}, "channel-publisher")
	mustStore(t, err)
	listing, err := store.CreateListing(ctx, creator, application.ListingInput{PublisherID: publisher.ID, Name: "Test Listing", ListingType: "workflow"}, "channel-listing")
	mustStore(t, err)
	listingVersion, err := store.CreateListingVersion(ctx, creator, listing.ID, application.ListingVersionInput{CapabilityManifest: map[string]any{"capability_ids": []any{capabilityItem.ID}}, PermissionManifest: map[string]any{"scopes": []any{"test.execute"}}, ContentRef: "registry://test-listing", Checksum: "sha256:test-listing-v1"}, "channel-listing-version")
	mustStore(t, err)
	for _, next := range []string{"submitted", "automated_review"} {
		listing, err = store.TransitionListing(ctx, creator, listing.ID, next, "channel-listing-"+next)
		mustStore(t, err)
	}
	if _, err = store.TransitionListing(ctx, creator, listing.ID, "manual_review", "channel-listing-bypass-auto-review"); err == nil {
		t.Fatal("listing bypassed automated review gate")
	}
	if _, err = store.ReviewListing(ctx, creator, listing.ID, application.ListingReviewInput{ListingVersionID: listingVersion.ID, ReviewType: "automated", Decision: "approved", Rationale: "self review"}, "channel-listing-self-review"); err == nil {
		t.Fatal("listing creator approved their own version")
	}
	_, err = store.ReviewListing(ctx, reviewer, listing.ID, application.ListingReviewInput{ListingVersionID: listingVersion.ID, ReviewType: "automated", Decision: "approved", Rationale: "Test automated policy passed"}, "channel-listing-auto-review")
	mustStore(t, err)
	listing, err = store.TransitionListing(ctx, creator, listing.ID, "manual_review", "channel-listing-manual")
	mustStore(t, err)
	for _, reviewType := range []string{"security", "license", "manual"} {
		_, err = store.ReviewListing(ctx, reviewer, listing.ID, application.ListingReviewInput{ListingVersionID: listingVersion.ID, ReviewType: reviewType, Decision: "approved", Rationale: "Test " + reviewType + " review passed"}, "channel-listing-review-"+reviewType)
		mustStore(t, err)
	}
	listing, err = store.TransitionListing(ctx, creator, listing.ID, "sandbox_testing", "channel-listing-sandbox")
	mustStore(t, err)
	_, err = store.RunSandbox(ctx, reviewer, application.SandboxRunInput{ListingVersionID: listingVersion.ID, Status: "succeeded", Policy: map[string]any{"network": "denied", "filesystem": "ephemeral"}, Result: map[string]any{"contract_tests": "passed"}}, "channel-sandbox-run")
	mustStore(t, err)
	_, err = store.RecordListingQuality(ctx, reviewer, application.ListingQualityInput{ListingVersionID: listingVersion.ID, ScoreBPS: 9200, Dimensions: map[string]any{"reliability": 9400, "security": 9000}}, "channel-listing-quality")
	mustStore(t, err)
	for _, next := range []string{"limited_release", "published"} {
		listing, err = store.TransitionListing(ctx, creator, listing.ID, next, "channel-listing-"+next)
		mustStore(t, err)
	}
	if listing.Status != "published" {
		t.Fatalf("listing status=%s", listing.Status)
	}

	dispute, err := store.CreateMarketplaceDispute(ctx, creator, application.MarketplaceDisputeInput{ListingID: listing.ID, ClaimantType: "platform", ClaimantID: "Test Platform", Reason: "Test dispute evidence"}, "channel-dispute")
	mustStore(t, err)
	dispute, err = store.ResolveMarketplaceDispute(ctx, reviewer, dispute.ID, application.DisputeResolutionInput{Decision: "resolved", Resolution: map[string]any{"outcome": "test remediation accepted"}}, "channel-dispute-resolve")
	mustStore(t, err)
	if dispute.Status != "resolved" {
		t.Fatalf("dispute status=%s", dispute.Status)
	}
	takedown, err := store.RequestTakedown(ctx, creator, application.TakedownInput{ListingID: listing.ID, Reason: "Test governed removal"}, "channel-takedown")
	mustStore(t, err)
	if _, err = store.ReviewTakedown(ctx, creator, takedown.ID, application.ReviewInput{Decision: "approved"}, "channel-takedown-self-review"); err == nil {
		t.Fatal("takedown requester approved their own request")
	}
	takedown, err = store.ReviewTakedown(ctx, reviewer, takedown.ID, application.ReviewInput{Decision: "approved", Rationale: "Test independent removal approval"}, "channel-takedown-review")
	mustStore(t, err)
	if takedown.Status != "executed" {
		t.Fatalf("takedown status=%s", takedown.Status)
	}

	overview, err := store.ListChannels(ctx, creator)
	mustStore(t, err)
	if len(overview.Resellers) != 2 || len(overview.LeadOwnerships) != 1 || len(overview.CustomerOwnerships) != 1 || len(overview.CommissionLocks) != 1 || len(overview.Suppliers) != 1 || len(overview.SupplierContracts) != 1 || len(overview.ProviderPayables) != 1 || len(overview.SupplierSettlements) != 1 || len(overview.ResellerSettlements) != 1 || len(overview.Marketplace.Listings) != 1 || overview.Marketplace.Listings[0].Status != "removed" {
		t.Fatalf("incomplete channel overview: %#v", overview)
	}
	otherOverview, err := store.ListChannels(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"})
	mustStore(t, err)
	if len(otherOverview.Resellers) != 0 || len(otherOverview.Suppliers) != 0 || len(otherOverview.Marketplace.Listings) != 0 {
		t.Fatalf("cross-tenant channel visibility: %#v", otherOverview)
	}
	if _, err = store.CreateCustomerOwnership(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}, application.CustomerOwnershipInput{CustomerID: "Test Customer", ResellerID: resellerB.ID, ProtectionDays: 30}, "channel-cross-tenant"); err == nil {
		t.Fatal("cross-tenant ownership assignment succeeded")
	}

	var audits, events int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND object_type IN ('lead_ownership','customer_ownership','transfer_request','commission_lock','supplier','supplier_contract','supplier_quality','listing','listing_review','sandbox_run','quality_score','marketplace_dispute','takedown')`, tenantID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_type IN ('lead_ownership','customer_ownership','transfer_request','commission_lock','supplier','supplier_contract','listing')`, tenantID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if audits < 24 || events < 24 {
		t.Fatalf("channel audit/outbox chain incomplete: audits=%d events=%d", audits, events)
	}
}

func seedCanonicalChannelFinance(t *testing.T, pool *pgxpool.Pool, tenantID, resellerID, providerID, endpointID string) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var priceBookID, orderID, chargeID, costID, debitAccountID, creditAccountID, postingID, settlementID, commissionID, payableID string
	if err = tx.QueryRow(ctx, `INSERT INTO price_books (tenant_id,currency,version,status) VALUES($1,'USD',1,'active') RETURNING id`, tenantID).Scan(&priceBookID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO orders (tenant_id,customer_id,status,currency,amount_minor,idempotency_key,version_bindings,created_by) VALUES($1,'Test Customer','completed','USD',700,'channel-fixture-order','{}','test') RETURNING id`, tenantID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO customer_charges (tenant_id,order_id,price_book_id,currency,amount_minor,idempotency_key,status,created_by) VALUES($1,$2,$3,'USD',700,'channel-fixture-charge','posted','test') RETURNING id`, tenantID, orderID, priceBookID).Scan(&chargeID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO provider_costs (tenant_id,order_id,provider_endpoint_id,currency,amount_minor,idempotency_key,created_by) VALUES($1,$2,$3,'USD',250,'channel-fixture-cost','test') RETURNING id`, tenantID, orderID, endpointID).Scan(&costID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO ledger_accounts (tenant_id,code,name,account_type,currency,purpose) VALUES($1,'channel-test-debit','Test Debit','expense','USD','test') RETURNING id`, tenantID).Scan(&debitAccountID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO ledger_accounts (tenant_id,code,name,account_type,currency,purpose) VALUES($1,'channel-test-credit','Test Credit','liability','USD','test') RETURNING id`, tenantID).Scan(&creditAccountID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO ledger_transactions (tenant_id,idempotency_key,reference_type,reference_id,description,transaction_type,created_by) VALUES($1,'channel-fixture-posting','channel_test','posting','Test canonical posting','adjustment','test') RETURNING id`, tenantID).Scan(&postingID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ledger_entries (tenant_id,transaction_id,account_id,direction,currency,amount_minor) VALUES($1,$2,$3,'debit','USD',320),($1,$2,$4,'credit','USD',320)`, tenantID, postingID, debitAccountID, creditAccountID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO commissions (tenant_id,customer_charge_id,beneficiary_type,beneficiary_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,'reseller',$3,'USD',70,70,'settled',$4,$5,'channel-fixture-commission','test') RETURNING id`, tenantID, chargeID, resellerID, creditAccountID, postingID).Scan(&commissionID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO provider_payables (tenant_id,provider_cost_id,provider_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,'USD',250,250,'settled',$4,$5,'channel-fixture-payable','test') RETURNING id`, tenantID, costID, providerID, creditAccountID, postingID).Scan(&payableID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO ledger_transactions (tenant_id,idempotency_key,reference_type,reference_id,description,transaction_type,created_by) VALUES($1,'channel-fixture-settlement','channel_test','settlement','Test canonical settlement','settlement','test') RETURNING id`, tenantID).Scan(&settlementID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ledger_entries (tenant_id,transaction_id,account_id,direction,currency,amount_minor) VALUES($1,$2,$3,'debit','USD',320),($1,$2,$4,'credit','USD',320)`, tenantID, settlementID, creditAccountID, debitAccountID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO settlements (tenant_id,source_type,source_id,beneficiary_type,beneficiary_id,currency,amount_minor,ledger_transaction_id,idempotency_key,created_by) VALUES($1,'commission',$2,'reseller',$3,'USD',70,$4,'channel-fixture-reseller-settlement','test'),($1,'provider_payable',$5,'provider',$6,'USD',250,$4,'channel-fixture-provider-settlement','test')`, tenantID, commissionID, resellerID, settlementID, payableID, providerID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return commissionID
}

func cleanupChannelTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{
		"takedowns", "marketplace_disputes", "payout_reserves", "revenue_share_rules", "incident_records", "quality_scores", "sandbox_runs", "listing_reviews", "listing_versions", "listings", "publishers", "developers",
		"supplier_quality_records", "supplier_rates", "supplier_contracts", "supplier_capabilities", "supplier_members",
		"settlement_cycles", "commission_locks", "commission_rules", "conflict_records", "transfer_requests", "customer_ownerships", "lead_ownerships", "attribution_rules", "reseller_members", "resellers", "reseller_levels",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE providers SET supplier_id=NULL WHERE tenant_id=$1`, tenantID); err != nil {
		t.Errorf("cleanup provider supplier binding: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM suppliers WHERE tenant_id=$1`, tenantID); err != nil {
		t.Errorf("cleanup suppliers: %v", err)
	}
	cleanupGrowthTenant(t, pool, tenantID)
}
