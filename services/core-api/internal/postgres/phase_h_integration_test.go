package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

func TestPhaseHNeutralPostgresEndToEnd(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Phase H")
	otherTenantID := createSecurityTenant(t, pool, "Phase H Other")
	defer cleanupPhaseHTenant(t, pool, tenantID)
	defer cleanupPhaseHTenant(t, pool, otherTenantID)

	store := NewStore(pool)
	operator := tenancy.Scope{TenantID: tenantID, ActorID: "phase-h-operator", Role: "operator", TraceID: "trace-phase-h"}
	reviewer := tenancy.Scope{TenantID: tenantID, ActorID: "phase-h-reviewer", Role: "reviewer", TraceID: "trace-phase-h-review"}
	admin := tenancy.Scope{TenantID: tenantID, ActorID: "phase-h-admin", Role: "admin", TraceID: "trace-phase-h-admin"}
	checkpoints := []string{"tenant"}

	source, err := store.CreateSource(ctx, operator, application.SourceInput{Name: "Test Source", ConnectorType: "manual", Config: map[string]any{"classification": "test"}}, "h-source")
	mustStore(t, err)
	signalInput := application.SignalInput{ExternalID: "test-signal-1", Payload: map[string]any{"summary": "Test neutral demand signal"}, Normalized: map[string]any{"entity": "Test Customer"}, OccurredAt: time.Now().UTC()}
	signal, err := store.ImportSignal(ctx, operator, source.ID, signalInput, "h-signal")
	mustStore(t, err)
	deduplicated, err := store.ImportSignal(ctx, operator, source.ID, signalInput, "h-signal-deduplicated")
	mustStore(t, err)
	if deduplicated.ID != signal.ID {
		t.Fatal("signal fingerprint deduplication created a second signal")
	}
	opp, err := store.PromoteSignal(ctx, operator, signal.ID, application.SignalPromotionInput{Name: "Test Opportunity", Description: "Neutral opportunity", Summary: "Test evidence from normalized signal", Confidence: 90}, "h-signal-promote")
	mustStore(t, err)
	checkpoints = append(checkpoints, "source", "signal", "opportunity", "evidence")
	opp, err = store.ScoreOpportunity(ctx, operator, opp.ID, 84, "h-opportunity-score")
	mustStore(t, err)
	opp, err = store.TransitionOpportunity(ctx, operator, opp.ID, "under_review", "h-opportunity-submit")
	mustStore(t, err)
	opp, err = store.ReviewOpportunity(ctx, reviewer, opp.ID, "approved", "Test evidence and score verified", "h-opportunity-review")
	mustStore(t, err)
	incubation, err := store.CreateIncubation(ctx, operator, opp.ID, "Test Incubation Project", "h-incubation")
	mustStore(t, err)
	for index, next := range []string{"researching", "validating", "approved", "building"} {
		incubation, err = store.TransitionIncubation(ctx, operator, incubation.ID, next, fmt.Sprintf("h-incubation-%d", index))
		mustStore(t, err)
	}
	blueprint, err := store.CreateBlueprint(ctx, operator, opp.ID, application.BlueprintInput{
		Name: "Test Business Blueprint", Description: "Neutral phase H blueprint", ValueProposition: "Test Value Proposition",
		RequiredCapabilities: []string{"Test Capability"}, ProductDefinitions: []map[string]any{{"name": "Test Product"}},
		WorkflowDefinitions: []map[string]any{{"name": "Test Workflow"}}, MeteringDefinitions: []map[string]any{{"unit": "test_unit"}},
		PricingDefinitions: []map[string]any{{"currency": "USD"}}, ComplianceProfile: map[string]any{"classification": "test"},
	}, "h-blueprint")
	mustStore(t, err)
	for index, next := range []string{"analyzing", "validating", "approved"} {
		blueprint, err = store.TransitionBlueprint(ctx, operator, blueprint.ID, next, fmt.Sprintf("h-blueprint-%d", index))
		mustStore(t, err)
	}
	checkpoints = append(checkpoints, "score", "review", "incubation", "blueprint")

	capabilityItem, err := store.CreateCapability(ctx, operator, "Test Capability", "Neutral test capability", map[string]any{"fixture": true}, "h-capability")
	mustStore(t, err)
	provider, err := store.CreateProvider(ctx, operator, "Test Provider", "h-provider")
	mustStore(t, err)
	endpoint, err := store.CreateProviderEndpoint(ctx, operator, provider.ID, capabilityItem.ID, "mock_realtime", "v1", "h-provider-endpoint")
	mustStore(t, err)
	supplier, err := store.CreateSupplier(ctx, operator, application.SupplierInput{Name: "Test Supplier"}, "h-supplier")
	mustStore(t, err)
	_, err = store.BindSupplierCapability(ctx, operator, application.SupplierCapabilityInput{SupplierID: supplier.ID, CapabilityID: capabilityItem.ID}, "h-supplier-capability")
	mustStore(t, err)
	_, err = store.BindProviderSupplier(ctx, operator, provider.ID, application.ProviderSupplierInput{SupplierID: supplier.ID}, "h-provider-supplier")
	mustStore(t, err)
	contract, err := store.CreateSupplierContract(ctx, operator, application.SupplierContractInput{SupplierID: supplier.ID, ProviderID: provider.ID, Name: "Test Supplier Contract", Currency: "USD", Terms: map[string]any{"settlement_days": 30}}, "h-supplier-contract")
	mustStore(t, err)
	contract, err = store.TransitionSupplierContract(ctx, operator, contract.ID, "pending_approval", "h-supplier-contract-submit")
	mustStore(t, err)
	contract, err = store.ReviewSupplierContract(ctx, reviewer, contract.ID, application.ReviewInput{Decision: "approved", Rationale: "Test terms verified"}, "h-supplier-contract-review")
	mustStore(t, err)
	_, err = store.CreateSupplierRate(ctx, operator, application.SupplierRateInput{ContractID: contract.ID, CapabilityID: capabilityItem.ID, Unit: "test_unit", RateMinor: 125}, "h-supplier-rate")
	mustStore(t, err)
	contract, err = store.TransitionSupplierContract(ctx, operator, contract.ID, "active", "h-supplier-contract-active")
	mustStore(t, err)
	_, err = store.RecordSupplierQuality(ctx, operator, application.SupplierQualityInput{SupplierID: supplier.ID, ProviderID: provider.ID, ProviderEndpointID: endpoint.ID, Metric: "test_success_rate", ScoreBPS: 9500, Evidence: map[string]any{"sample_size": 10}, PeriodStart: time.Now().UTC().AddDate(0, 0, -30), PeriodEnd: time.Now().UTC()}, "h-supplier-quality")
	mustStore(t, err)
	checkpoints = append(checkpoints, "capability", "provider", "supplier", "supplier_contract")

	product, err := store.CreateProduct(ctx, operator, blueprint.ID, "Test Product", "h-product")
	mustStore(t, err)
	versionInput := neutralProductVersionInput(capabilityItem.ID)
	versionInput.Workflow.Nodes = append(versionInput.Workflow.Nodes, workflow.Node{ID: "verify", Type: workflow.RealtimeCall})
	for index, edge := range versionInput.Workflow.Edges {
		if edge.From == "execute" && edge.To == "meter" {
			versionInput.Workflow.Edges[index].To = "verify"
		}
	}
	versionInput.Workflow.Edges = append(versionInput.Workflow.Edges, workflow.Edge{From: "verify", To: "meter"})
	productVersion, err := store.CreateProductVersion(ctx, operator, product.ID, versionInput, "h-product-version")
	mustStore(t, err)
	sku, err := store.CreateSKU(ctx, operator, product.ID, "TEST-H-SKU", "Test SKU", "h-sku")
	mustStore(t, err)
	skuVersion, err := store.CreateSKUVersion(ctx, operator, sku.ID, application.SKUVersionInput{ProductVersionID: productVersion.ID, Entitlements: map[string]any{"test_limit": 10}}, "h-sku-version")
	mustStore(t, err)
	_, err = store.PublishProduct(ctx, operator, product.ID, productVersion.ID, "h-publication")
	mustStore(t, err)
	checkpoints = append(checkpoints, "product", "sku", "workflow_definition", "metering", "pricing", "routing", "publication")

	segment, err := store.CreateMarketSegment(ctx, operator, application.MarketSegmentInput{Name: "Test Market Segment", Definition: map[string]any{"fixture": true}}, "h-segment")
	mustStore(t, err)
	icp, err := store.CreateICPDefinition(ctx, operator, segment.ID, application.ICPDefinitionInput{Name: "Test ICP", Definition: map[string]any{"signals": []any{"test"}}}, "h-icp")
	mustStore(t, err)
	lead, err := store.CreateLead(ctx, operator, application.LeadInput{MarketSegmentID: segment.ID, ICPDefinitionID: icp.ID, Name: "Test Customer"}, "h-lead")
	mustStore(t, err)
	_, err = store.AddLeadEvidence(ctx, operator, lead.ID, application.LeadEvidenceInput{Kind: "test_signal", Summary: "Test demand evidence", Confidence: 90, SourceRef: "signal:" + signal.ID}, "h-lead-evidence")
	mustStore(t, err)
	lead, err = store.TransitionLead(ctx, operator, lead.ID, "qualified", "h-lead-qualified")
	mustStore(t, err)
	proofTemplate, err := store.CreateProofTemplate(ctx, operator, application.ProofTemplateInput{Name: "Test Proof Template", ProofType: "analysis", WorkflowVersionID: productVersion.Workflow.ID, InputSchema: neutralProductVersionInput(capabilityItem.ID).InputSchema, OutputSchema: neutralProductVersionInput(capabilityItem.ID).OutputSchema, AccessPolicy: map[string]any{"scope": "tenant"}, RetentionDays: 30}, "h-proof-template")
	mustStore(t, err)
	proofRequest, err := store.CreateProofRequest(ctx, operator, lead.ID, application.ProofRequestInput{TemplateID: proofTemplate.ID, Input: map[string]any{"fixture": true}}, "h-proof-request")
	mustStore(t, err)
	_, err = store.GenerateProof(ctx, operator, proofRequest.ID, application.ProofGenerationInput{Result: map[string]any{"summary": "Test Proof Artifact"}, ArtifactRef: "proof://test-artifact"}, "h-proof-generate")
	mustStore(t, err)
	_, err = store.ReviewProof(ctx, reviewer, proofRequest.ID, application.ProofReviewInput{Decision: "approved", Rationale: "Test proof verified"}, "h-proof-review")
	mustStore(t, err)
	for index, next := range []string{"approved_for_outreach", "contacted", "replied", "meeting"} {
		lead, err = store.TransitionLead(ctx, operator, lead.ID, next, fmt.Sprintf("h-lead-%d", index))
		mustStore(t, err)
	}
	campaign, err := store.CreateCampaign(ctx, operator, application.CampaignInput{MarketSegmentID: segment.ID, Name: "Test Campaign", Channel: "email", Purpose: "Internal test planning only"}, "h-campaign")
	mustStore(t, err)
	_, err = store.AddCampaignStep(ctx, operator, campaign.ID, application.CampaignStepInput{Position: 1, Kind: "message", Definition: map[string]any{"template": "test"}}, "h-campaign-step")
	mustStore(t, err)
	campaign, err = store.TransitionCampaign(ctx, operator, campaign.ID, "pending_approval", "h-campaign-submit")
	mustStore(t, err)
	campaign, err = store.ReviewCampaign(ctx, reviewer, campaign.ID, application.CampaignApprovalInput{Decision: "approved", Rationale: "Internal plan verified; delivery remains disabled"}, "h-campaign-review")
	mustStore(t, err)
	deal, err := store.CreateDeal(ctx, operator, application.DealInput{LeadID: lead.ID, Name: "Test Deal", CustomerID: "Test Customer", Currency: "USD", ValueMinor: 700}, "h-deal")
	mustStore(t, err)
	checkpoints = append(checkpoints, "segment", "lead", "proof", "campaign", "deal")

	level, err := store.CreateResellerLevel(ctx, operator, application.ResellerLevelInput{Name: "Test Reseller Level", Rank: 1, DefaultCommissionBPS: 1000}, "h-reseller-level")
	mustStore(t, err)
	reseller, err := store.CreateReseller(ctx, operator, application.ResellerInput{LevelID: level.ID, Name: "Test Reseller"}, "h-reseller")
	mustStore(t, err)
	attribution, err := store.CreateAttributionRule(ctx, operator, application.AttributionRuleInput{Name: "Test Attribution Rule", Priority: 1, Definition: map[string]any{"method": "first_verified_evidence"}}, "h-attribution")
	mustStore(t, err)
	leadOwnership, err := store.AssignLeadOwnership(ctx, operator, application.LeadOwnershipInput{LeadID: lead.ID, ResellerID: reseller.ID, AttributionRuleID: attribution.ID, ProtectionDays: 30}, "h-lead-ownership")
	mustStore(t, err)
	_, err = store.CreateCustomerOwnership(ctx, operator, application.CustomerOwnershipInput{CustomerID: "Test Customer", ResellerID: reseller.ID, SourceLeadOwnershipID: leadOwnership.ID, ProtectionDays: 90}, "h-customer-ownership")
	mustStore(t, err)
	commissionRule, err := store.CreateCommissionRule(ctx, operator, application.CommissionRuleInput{Name: "Test Commission Rule", ResellerID: reseller.ID, BasisPoints: 1000, EffectiveFrom: time.Now().UTC().Add(-time.Hour)}, "h-commission-rule")
	mustStore(t, err)
	checkpoints = append(checkpoints, "reseller", "ownership")

	quote, err := store.CreateQuote(ctx, operator, application.QuoteInput{DealID: deal.ID, CustomerID: deal.CustomerID, Currency: "USD", ValidUntil: time.Now().UTC().Add(24 * time.Hour), Items: []application.QuoteItemInput{{SKUVersionID: skuVersion.ID, Quantity: 2, Input: map[string]any{"input": "Test Order Input"}}}}, "h-quote")
	mustStore(t, err)
	deal, err = store.TransitionDeal(ctx, operator, deal.ID, "proposal", "h-deal-proposal")
	mustStore(t, err)
	quote, err = store.TransitionQuote(ctx, operator, quote.ID, "accepted", "h-quote-accepted")
	mustStore(t, err)
	deal, err = store.TransitionDeal(ctx, operator, deal.ID, "won", "h-deal-won")
	mustStore(t, err)
	order, err := store.CreateOrder(ctx, operator, quote.Versions[0].ID, "h-order")
	mustStore(t, err)
	order, err = store.TransitionOrder(ctx, operator, order.ID, "awaiting_payment", "h-order-awaiting-payment")
	mustStore(t, err)
	wallet, err := store.CreateWallet(ctx, operator, application.WalletInput{OwnerType: "customer", OwnerID: "Test Customer", Currency: "USD"}, "h-wallet")
	mustStore(t, err)
	_, err = store.PostWalletAdjustment(ctx, admin, wallet.ID, application.WalletAdjustmentInput{Direction: "credit", AmountMinor: 1200, Reason: "Test funding"}, "h-wallet-funding")
	mustStore(t, err)
	hold, err := store.PlaceOrderHold(ctx, operator, order.ID, application.HoldInput{WalletID: wallet.ID, AmountMinor: 800}, "h-hold")
	mustStore(t, err)
	for _, next := range []string{"paid", "provisioning"} {
		order, err = store.TransitionOrder(ctx, operator, order.ID, next, "h-order-"+next)
		mustStore(t, err)
	}
	execution := order.Executions[0]
	for _, transition := range []application.ExecutionTransitionInput{{To: "validating"}, {To: "reserved", ProviderEndpointID: endpoint.ID}, {To: "queued"}} {
		execution, err = store.TransitionExecution(ctx, operator, execution.ID, transition, "h-execution-"+transition.To)
		mustStore(t, err)
	}
	checkpoints = append(checkpoints, "quote", "order", "hold")

	secretRef, secret := "OPPORTUNITY_ADAPTER_SECRET_PHASE_H_TEST", "phase-h-test-secret-material-32-bytes"
	t.Setenv(secretRef, secret)
	identity, err := store.RegisterAdapterIdentity(ctx, admin, application.AdapterIdentityInput{Name: "Test Adapter Identity", KeyID: "phase-h-test-adapter", ProviderEndpointID: endpoint.ID, SecretRef: secretRef}, "h-adapter-identity")
	mustStore(t, err)
	run, err := store.StartWorkflowRun(ctx, operator, execution.ID, application.WorkflowRunInput{MaxAttempts: 3}, "h-workflow-run")
	mustStore(t, err)
	step, err := store.LeaseWorkflowStep(ctx, operator, application.WorkflowLeaseInput{AdapterIdentityID: identity.ID, LeaseSeconds: 120}, "h-workflow-lease")
	mustStore(t, err)
	if step.RunID != run.ID || step.ExecutionOrderID != execution.ID {
		t.Fatalf("workflow lease is not bound to the execution: %#v", step)
	}
	resultBody, err := json.Marshal(map[string]any{"external_event_id": "test-adapter-result-1", "execution_id": execution.ID, "status": "succeeded", "external_id": "test-external-execution", "output": map[string]any{"result": "Test Result", "units": 2}})
	mustStore(t, err)
	timestamp, nonce := strconv.FormatInt(time.Now().UTC().Unix(), 10), "test-nonce-phase-h-1"
	request := application.AdapterIngressRequest{KeyID: identity.KeyID, Timestamp: timestamp, Nonce: nonce, Body: resultBody}
	request.Signature = adapterSignature(secret, timestamp, nonce, resultBody)
	invalid := request
	invalid.Signature = "invalid"
	if _, err = store.IngestAdapterResult(ctx, invalid); err == nil {
		t.Fatal("invalid Adapter signature was accepted")
	}
	receipt, err := store.IngestAdapterResult(ctx, request)
	mustStore(t, err)
	replayedReceipt, err := store.IngestAdapterResult(ctx, request)
	mustStore(t, err)
	if replayedReceipt.ID != receipt.ID {
		t.Fatal("Adapter event replay created a second receipt")
	}
	order, err = store.GetOrder(ctx, operator, order.ID)
	mustStore(t, err)
	execution = order.Executions[0]
	if execution.Status != "processing" {
		t.Fatalf("one remote step completed the multi-step execution early: %#v", execution)
	}
	secondStep, err := store.LeaseWorkflowStep(ctx, operator, application.WorkflowLeaseInput{AdapterIdentityID: identity.ID, LeaseSeconds: 120}, "h-workflow-lease-second")
	mustStore(t, err)
	if secondStep.RunID != run.ID || secondStep.ID == step.ID {
		t.Fatalf("second workflow step was not independently leased: %#v", secondStep)
	}
	secondBody, err := json.Marshal(map[string]any{"external_event_id": "test-adapter-result-2", "execution_id": execution.ID, "status": "succeeded", "external_id": "test-external-execution", "output": map[string]any{"result": "Test Result Verified", "units": 2}})
	mustStore(t, err)
	secondTimestamp, secondNonce := strconv.FormatInt(time.Now().UTC().Unix(), 10), "test-nonce-phase-h-2"
	_, err = store.IngestAdapterResult(ctx, application.AdapterIngressRequest{KeyID: identity.KeyID, Timestamp: secondTimestamp, Nonce: secondNonce, Signature: adapterSignature(secret, secondTimestamp, secondNonce, secondBody), Body: secondBody})
	mustStore(t, err)
	order, err = store.GetOrder(ctx, operator, order.ID)
	mustStore(t, err)
	execution = order.Executions[0]
	if execution.Status != "succeeded" || execution.ExternalID != "test-external-execution" || execution.Attempt != 1 {
		t.Fatalf("trusted Adapter result did not advance execution: %#v", execution)
	}
	checkpoints = append(checkpoints, "workflow_lease", "adapter_receipt", "execution")

	usage, err := store.RecordUsage(ctx, operator, execution.ID, 2, time.Now().UTC(), "h-usage")
	mustStore(t, err)
	providerCost, err := store.RecordProviderCost(ctx, operator, execution.ID, endpoint.ID, "USD", 250, "h-provider-cost")
	mustStore(t, err)
	charge, err := store.CreateCustomerCharge(ctx, operator, execution.ID, "h-customer-charge")
	mustStore(t, err)
	_, err = store.PostCustomerCharge(ctx, operator, charge.ID, application.ChargePostingInput{HoldID: hold.ID}, "h-charge-post")
	mustStore(t, err)
	_, err = store.ReleaseHold(ctx, operator, hold.ID, application.ReleaseInput{}, "h-hold-release")
	mustStore(t, err)
	payable, err := store.CreateProviderPayable(ctx, operator, providerCost.ID, "h-provider-payable")
	mustStore(t, err)
	commission, err := store.CreateCommission(ctx, operator, charge.ID, application.CommissionInput{BeneficiaryType: "reseller", BeneficiaryID: reseller.ID, AmountMinor: 70}, "h-commission")
	mustStore(t, err)
	_, err = store.LockCommission(ctx, operator, application.CommissionLockInput{CommissionID: commission.ID, CommissionRuleID: commissionRule.ID, ResellerID: reseller.ID}, "h-commission-lock")
	mustStore(t, err)
	reconciliation, err := store.RunReconciliation(ctx, operator, application.ReconciliationInput{OrderID: order.ID}, "h-reconciliation")
	mustStore(t, err)
	if reconciliation.Status != "matched" {
		t.Fatalf("phase H reconciliation status=%s", reconciliation.Status)
	}
	_, err = store.CreateSettlement(ctx, operator, application.SettlementInput{SourceType: "provider_payable", SourceID: payable.ID}, "h-provider-settlement")
	mustStore(t, err)
	_, err = store.CreateSettlement(ctx, operator, application.SettlementInput{SourceType: "commission", SourceID: commission.ID}, "h-commission-settlement")
	mustStore(t, err)
	_, err = store.CreateSettlementCycle(ctx, operator, application.SettlementCycleInput{ResellerID: reseller.ID, Name: "Test Settlement Cycle", PeriodStart: time.Now().UTC().AddDate(0, 0, -7), PeriodEnd: time.Now().UTC()}, "h-settlement-cycle")
	mustStore(t, err)
	delivery := order.Deliveries[0]
	delivery, err = store.TransitionDelivery(ctx, operator, delivery.ID, "in_progress", "h-delivery-start")
	mustStore(t, err)
	_, err = store.TransitionDelivery(ctx, operator, delivery.ID, "completed", "h-delivery-complete")
	mustStore(t, err)
	order, err = store.TransitionOrder(ctx, operator, order.ID, "active", "h-order-active")
	mustStore(t, err)
	execution, err = store.TransitionExecution(ctx, operator, execution.ID, application.ExecutionTransitionInput{To: "reconciling"}, "h-execution-reconciling")
	mustStore(t, err)
	execution, err = store.TransitionExecution(ctx, operator, execution.ID, application.ExecutionTransitionInput{To: "settled"}, "h-execution-settled")
	mustStore(t, err)
	order, err = store.TransitionOrder(ctx, operator, order.ID, "completed", "h-order-completed")
	mustStore(t, err)
	checkpoints = append(checkpoints, "usage", "provider_cost", "customer_charge", "ledger", "commission", "payable", "settlement", "reconciliation")

	developer, err := store.CreateDeveloper(ctx, operator, application.DeveloperInput{Name: "Test Developer"}, "h-developer")
	mustStore(t, err)
	publisher, err := store.CreatePublisher(ctx, operator, application.PublisherInput{DeveloperID: developer.ID, Name: "Test Publisher"}, "h-publisher")
	mustStore(t, err)
	listing, err := store.CreateListing(ctx, operator, application.ListingInput{PublisherID: publisher.ID, Name: "Test Listing", ListingType: "workflow"}, "h-listing")
	mustStore(t, err)
	listingVersion, err := store.CreateListingVersion(ctx, operator, listing.ID, application.ListingVersionInput{CapabilityManifest: map[string]any{"capability_ids": []any{capabilityItem.ID}}, PermissionManifest: map[string]any{"scopes": []any{"test.execute"}}, ContentRef: "registry://test-listing", Checksum: "sha256:phase-h-test-listing"}, "h-listing-version")
	mustStore(t, err)
	for _, next := range []string{"submitted", "automated_review"} {
		listing, err = store.TransitionListing(ctx, operator, listing.ID, next, "h-listing-"+next)
		mustStore(t, err)
	}
	_, err = store.ReviewListing(ctx, reviewer, listing.ID, application.ListingReviewInput{ListingVersionID: listingVersion.ID, ReviewType: "automated", Decision: "approved", Rationale: "Test policy passed"}, "h-listing-auto-review")
	mustStore(t, err)
	listing, err = store.TransitionListing(ctx, operator, listing.ID, "manual_review", "h-listing-manual-review")
	mustStore(t, err)
	for _, reviewType := range []string{"security", "license", "manual"} {
		_, err = store.ReviewListing(ctx, reviewer, listing.ID, application.ListingReviewInput{ListingVersionID: listingVersion.ID, ReviewType: reviewType, Decision: "approved", Rationale: "Test " + reviewType + " review passed"}, "h-listing-review-"+reviewType)
		mustStore(t, err)
	}
	listing, err = store.TransitionListing(ctx, operator, listing.ID, "sandbox_testing", "h-listing-sandbox")
	mustStore(t, err)
	_, err = store.RunSandbox(ctx, reviewer, application.SandboxRunInput{ListingVersionID: listingVersion.ID, Status: "succeeded", Policy: map[string]any{"network": "denied"}, Result: map[string]any{"contract_tests": "passed"}}, "h-sandbox-run")
	mustStore(t, err)
	_, err = store.RecordListingQuality(ctx, reviewer, application.ListingQualityInput{ListingVersionID: listingVersion.ID, ScoreBPS: 9200, Dimensions: map[string]any{"reliability": 9400}}, "h-listing-quality")
	mustStore(t, err)
	for _, next := range []string{"limited_release", "published"} {
		listing, err = store.TransitionListing(ctx, operator, listing.ID, next, "h-listing-"+next)
		mustStore(t, err)
	}
	checkpoints = append(checkpoints, "marketplace_listing", "review", "sandbox")

	feedback, err := store.RecordOutcomeFeedback(ctx, operator, application.OutcomeFeedbackInput{OpportunityID: opp.ID, OrderID: order.ID, ExecutionOrderID: execution.ID, Metrics: map[string]any{"conversion": "won"}, Evidence: map[string]any{"reconciliation_run_id": reconciliation.ID}}, "h-outcome-feedback")
	mustStore(t, err)
	if feedback.Metrics["gross_margin_minor"] != float64(380) {
		t.Fatalf("unexpected outcome margin: %#v", feedback.Metrics)
	}
	analyticsOverview, err := store.ListAnalytics(ctx, operator)
	mustStore(t, err)
	if len(analyticsOverview.Feedback) != 1 || len(analyticsOverview.Projections) != 1 || analyticsOverview.Projections[0].FeedbackCount != 1 {
		t.Fatalf("outcome projection is incomplete: %#v", analyticsOverview)
	}
	checkpoints = append(checkpoints, "outcome_feedback", "opportunity_projection")

	var deadEventID string
	if err = pool.QueryRow(ctx, `INSERT INTO outbox_events (tenant_id,aggregate_type,aggregate_id,event_type,aggregate_version,trace_id,payload,locked_by,locked_until) VALUES($1,'test','phase-h','test.dead_letter',1,'trace-phase-h','{}','phase-h-worker',now()+interval '1 minute') RETURNING id`, tenantID).Scan(&deadEventID); err != nil {
		t.Fatal(err)
	}
	mustStore(t, store.MarkOutboxFailed(ctx, "phase-h-worker", deadEventID, time.Time{}, "Test delivery failure", "Test attempts exhausted"))
	operationsOverview, err := store.ListOperations(ctx, admin)
	mustStore(t, err)
	if operationsOverview.Outbox.DeadLetter != 1 || len(operationsOverview.Alerts) != 1 || len(operationsOverview.AdapterReceipts) != 2 || len(operationsOverview.WorkflowRuns) != 1 {
		t.Fatalf("phase H operations overview is incomplete: %#v", operationsOverview)
	}
	for _, check := range operationsOverview.DeploymentChecks {
		if check.Status != "passed" {
			t.Fatalf("deployment check %s failed: %s", check.Name, check.Message)
		}
	}
	_, err = store.ReplayOutbox(ctx, admin, deadEventID, "Test operator verified dependency recovery", "h-outbox-replay")
	mustStore(t, err)
	operationsOverview, err = store.ListOperations(ctx, admin)
	mustStore(t, err)
	if operationsOverview.Outbox.DeadLetter != 0 || operationsOverview.Alerts[0].Status != "resolved" {
		t.Fatalf("dead-letter replay did not resolve operational state: %#v", operationsOverview)
	}
	checkpoints = append(checkpoints, "operations", "outbox_replay")

	otherIntelligence, err := store.ListIntelligence(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"})
	mustStore(t, err)
	otherAnalytics, err := store.ListAnalytics(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"})
	mustStore(t, err)
	if len(otherIntelligence.Sources) != 0 || len(otherIntelligence.Signals) != 0 || len(otherAnalytics.Feedback) != 0 {
		t.Fatal("phase H facts crossed tenant boundaries")
	}
	var auditCount, outboxCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1`, tenantID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1`, tenantID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if len(checkpoints) < 45 || auditCount < 70 || outboxCount < 60 || usage.Quantity != 2 || contract.Status != "active" || listing.Status != "published" || order.Status != "completed" {
		t.Fatalf("phase H chain incomplete: checkpoints=%d audit=%d outbox=%d", len(checkpoints), auditCount, outboxCount)
	}
}

func cleanupPhaseHTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{
		"outcome_feedback", "adapter_result_receipts", "workflow_step_runs", "workflow_runs", "adapter_identities",
		"outbox_replays", "operational_alerts", "opportunity_signals", "signals", "sources",
		"takedowns", "marketplace_disputes", "payout_reserves", "revenue_share_rules", "incident_records", "quality_scores", "sandbox_runs", "listing_reviews", "listing_versions", "listings", "publishers", "developers",
		"supplier_quality_records", "supplier_rates", "supplier_contracts", "supplier_capabilities", "supplier_members",
		"settlement_cycles", "commission_locks", "commission_rules", "conflict_records", "transfer_requests", "customer_ownerships", "lead_ownerships", "attribution_rules", "reseller_members", "resellers", "reseller_levels",
		"reconciliation_discrepancies", "reconciliation_items", "reconciliation_runs", "settlements", "settlement_reserves", "chargeback_reserves", "promotional_credits",
		"refunds", "commissions", "provider_payables", "hold_releases", "holds", "financial_adjustments", "payment_receivables", "balance_snapshots", "ledger_entries",
		"customer_charges", "provider_costs", "usage_records", "delivery_projects", "execution_orders", "entitlements", "subscriptions", "order_items", "orders",
		"ledger_transactions", "ledger_accounts", "wallets",
		"conversation_messages", "conversations", "outreach_messages", "send_quotas", "suppression_entries", "campaign_approvals", "campaign_steps", "campaigns",
		"proof_instances", "proof_requests", "proof_templates", "experiments", "quote_version_items", "quote_versions", "quotes", "deals", "contact_sources", "contacts", "lead_evidence", "leads", "icp_definitions", "market_segments",
		"publications", "product_compliance_bindings", "product_growth_bindings", "product_output_definitions", "product_form_definitions", "product_routing_bindings", "product_pricing_bindings", "product_metering_bindings",
		"product_workflow_bindings", "product_capability_bindings", "sku_versions", "skus", "product_versions", "price_rules", "metering_definitions", "price_books", "route_policies", "products",
		"opportunity_reviews", "incubation_projects", "opportunity_evidence", "business_blueprints", "provider_endpoints",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup phase H %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE providers SET supplier_id=NULL WHERE tenant_id=$1`, tenantID); err != nil {
		t.Errorf("cleanup phase H provider supplier binding: %v", err)
	}
	for _, table := range []string{"suppliers", "providers", "capabilities", "workflow_definitions", "auth_sessions", "inbox_events", "tenant_features", "command_idempotency", "audit_log", "outbox_events", "opportunities", "memberships", "brands"} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup phase H %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, tenantID); err != nil {
		t.Errorf("cleanup phase H tenant: %v", err)
	}
}
