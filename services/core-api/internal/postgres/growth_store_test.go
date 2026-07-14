package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func TestGrowthControlPlanePersistence(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Growth")
	otherTenantID := createSecurityTenant(t, pool, "Growth Other")
	defer cleanupGrowthTenant(t, pool, tenantID)
	defer cleanupGrowthTenant(t, pool, otherTenantID)

	var opportunityID string
	if err := pool.QueryRow(ctx, `INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'Test Opportunity','incubating','test') RETURNING id`, tenantID).Scan(&opportunityID); err != nil {
		t.Fatal(err)
	}
	var blueprintID string
	if err := pool.QueryRow(ctx, `INSERT INTO business_blueprints(tenant_id,source_opportunity_id,name,version,status,definition,created_by,approved_by) VALUES($1,$2,'Test Business Blueprint',1,'approved','{}','test','test') RETURNING id`, tenantID, opportunityID).Scan(&blueprintID); err != nil {
		t.Fatal(err)
	}

	store := NewStore(pool)
	scope := tenancy.Scope{TenantID: tenantID, ActorID: "growth-test", TraceID: "trace-growth-test"}
	capabilityItem, err := store.CreateCapability(ctx, scope, "Test Capability", "Neutral growth fixture capability", map[string]any{"fixture": true}, "growth-capability")
	mustStore(t, err)
	provider, err := store.CreateProvider(ctx, scope, "Test Provider", "growth-provider")
	mustStore(t, err)
	_, err = store.CreateProviderEndpoint(ctx, scope, provider.ID, capabilityItem.ID, "mock_realtime", "v1", "growth-endpoint")
	mustStore(t, err)
	product, err := store.CreateProduct(ctx, scope, blueprintID, "Test Product", "growth-product")
	mustStore(t, err)
	productVersion, err := store.CreateProductVersion(ctx, scope, product.ID, neutralProductVersionInput(capabilityItem.ID), "growth-product-version")
	mustStore(t, err)
	sku, err := store.CreateSKU(ctx, scope, product.ID, "TEST-GROWTH-SKU", "Test SKU", "growth-sku")
	mustStore(t, err)
	skuVersion, err := store.CreateSKUVersion(ctx, scope, sku.ID, application.SKUVersionInput{ProductVersionID: productVersion.ID, Entitlements: map[string]any{"test_limit": 10}}, "growth-sku-version")
	mustStore(t, err)
	_, err = store.PublishProduct(ctx, scope, product.ID, productVersion.ID, "growth-publication")
	mustStore(t, err)

	segmentInput := application.MarketSegmentInput{Name: "Test Market Segment", Definition: map[string]any{"criteria": []any{"test"}}}
	segment, err := store.CreateMarketSegment(ctx, scope, segmentInput, "growth-segment")
	mustStore(t, err)
	replayedSegment, err := store.CreateMarketSegment(ctx, scope, segmentInput, "growth-segment")
	mustStore(t, err)
	if replayedSegment.ID != segment.ID {
		t.Fatal("market segment idempotency replay created a second record")
	}
	icp, err := store.CreateICPDefinition(ctx, scope, segment.ID, application.ICPDefinitionInput{Name: "Test ICP", Definition: map[string]any{"signals": []any{"test"}}}, "growth-icp")
	mustStore(t, err)

	lead, err := store.CreateLead(ctx, scope, application.LeadInput{MarketSegmentID: segment.ID, ICPDefinitionID: icp.ID, Name: "Test Customer"}, "growth-lead")
	mustStore(t, err)
	_, err = store.AddLeadEvidence(ctx, scope, lead.ID, application.LeadEvidenceInput{Kind: "test_signal", Summary: "Test evidence", Confidence: 85, SourceRef: "test://evidence"}, "growth-evidence")
	mustStore(t, err)
	lead, err = store.TransitionLead(ctx, scope, lead.ID, "qualified", "growth-lead-qualified")
	mustStore(t, err)
	contact, err := store.CreateContact(ctx, scope, lead.ID, application.ContactInput{Channel: "email", Value: "test.customer@example.invalid", ConsentStatus: "unknown", SourceType: "manual", SourceRef: "test", Evidence: map[string]any{"fixture": true}}, "growth-contact")
	mustStore(t, err)

	proofTemplate, err := store.CreateProofTemplate(ctx, scope, application.ProofTemplateInput{
		Name: "Test Proof Template", ProofType: "analysis", WorkflowVersionID: productVersion.Workflow.ID,
		InputSchema: schema.Definition{"type": "object", "properties": map[string]any{}}, OutputSchema: schema.Definition{"type": "object", "properties": map[string]any{}},
		AccessPolicy: map[string]any{"scope": "tenant"}, RetentionDays: 30,
	}, "growth-proof-template")
	mustStore(t, err)
	proofRequest, err := store.CreateProofRequest(ctx, scope, lead.ID, application.ProofRequestInput{TemplateID: proofTemplate.ID, Input: map[string]any{"fixture": true}}, "growth-proof-request")
	mustStore(t, err)
	proofInstance, err := store.GenerateProof(ctx, scope, proofRequest.ID, application.ProofGenerationInput{Result: map[string]any{"summary": "Test Proof Artifact"}, ArtifactRef: "proof://test-artifact"}, "growth-proof-generate")
	mustStore(t, err)
	proofInstance, err = store.ReviewProof(ctx, scope, proofRequest.ID, application.ProofReviewInput{Decision: "approved", Rationale: "Test proof verified"}, "growth-proof-review")
	mustStore(t, err)
	if proofInstance.Status != "approved" {
		t.Fatalf("proof status=%s", proofInstance.Status)
	}
	lead, err = store.TransitionLead(ctx, scope, lead.ID, "approved_for_outreach", "growth-lead-approved-outreach")
	mustStore(t, err)

	campaign, err := store.CreateCampaign(ctx, scope, application.CampaignInput{MarketSegmentID: segment.ID, Name: "Test Campaign", Channel: "email", Purpose: "Test approved outreach planning"}, "growth-campaign")
	mustStore(t, err)
	step, err := store.AddCampaignStep(ctx, scope, campaign.ID, application.CampaignStepInput{Position: 1, Kind: "message", Definition: map[string]any{"template": "test"}}, "growth-campaign-step")
	mustStore(t, err)
	campaign, err = store.TransitionCampaign(ctx, scope, campaign.ID, "pending_approval", "growth-campaign-submit")
	mustStore(t, err)
	if _, err = store.TransitionCampaign(ctx, scope, campaign.ID, "active", "growth-campaign-activate-without-review"); err == nil {
		t.Fatal("campaign activated without persisted approval")
	}
	campaign, err = store.ReviewCampaign(ctx, scope, campaign.ID, application.CampaignApprovalInput{Decision: "approved", Rationale: "Test campaign controls verified"}, "growth-campaign-review")
	mustStore(t, err)
	campaign, err = store.TransitionCampaign(ctx, scope, campaign.ID, "active", "growth-campaign-activate")
	mustStore(t, err)

	_, err = store.CreateSuppression(ctx, scope, application.SuppressionInput{ContactID: contact.ID, Channel: "email", Reason: "do_not_contact", SourceRef: "test"}, "growth-suppression")
	mustStore(t, err)
	blocked, err := store.PlanOutreach(ctx, scope, campaign.ID, application.OutreachPlanInput{LeadID: lead.ID, StepID: step.ID, ContactID: contact.ID, Content: map[string]any{"subject": "Test", "body": "Test"}}, "growth-outreach-blocked")
	mustStore(t, err)
	if blocked.Status != "blocked" || blocked.BlockReason != "suppressed" {
		t.Fatalf("suppressed outreach was not audibly blocked: %#v", blocked)
	}

	leadTwo, err := store.CreateLead(ctx, scope, application.LeadInput{MarketSegmentID: segment.ID, ICPDefinitionID: icp.ID, Name: "Test Customer Two"}, "growth-lead-two")
	mustStore(t, err)
	_, err = store.AddLeadEvidence(ctx, scope, leadTwo.ID, application.LeadEvidenceInput{Kind: "test_signal", Summary: "Test second evidence", Confidence: 90}, "growth-evidence-two")
	mustStore(t, err)
	leadTwo, err = store.TransitionLead(ctx, scope, leadTwo.ID, "qualified", "growth-lead-two-qualified")
	mustStore(t, err)
	leadTwo, err = store.TransitionLead(ctx, scope, leadTwo.ID, "approved_for_outreach", "growth-lead-two-approved-outreach")
	mustStore(t, err)
	contactTwo, err := store.CreateContact(ctx, scope, leadTwo.ID, application.ContactInput{Channel: "email", Value: "test.customer.two@example.invalid", ConsentStatus: "opted_in", SourceType: "manual", SourceRef: "test", Evidence: map[string]any{"fixture": true}}, "growth-contact-two")
	mustStore(t, err)
	planned, err := store.PlanOutreach(ctx, scope, campaign.ID, application.OutreachPlanInput{LeadID: leadTwo.ID, StepID: step.ID, ContactID: contactTwo.ID, Content: map[string]any{"subject": "Test", "body": "Test"}}, "growth-outreach-planned")
	mustStore(t, err)
	if planned.Status != "planned" {
		t.Fatalf("outreach was not planned: %#v", planned)
	}
	planned, err = store.TransitionOutreach(ctx, scope, planned.ID, application.OutreachTransitionInput{To: "cancelled"}, "growth-outreach-cancel")
	mustStore(t, err)
	if planned.Status != "cancelled" {
		t.Fatalf("planned outreach was not cancelled: %#v", planned)
	}
	var reservedCount int
	if err = pool.QueryRow(ctx, `SELECT reserved_count FROM send_quotas WHERE tenant_id=$1 AND channel='email'`, tenantID).Scan(&reservedCount); err != nil || reservedCount != 0 {
		t.Fatalf("quota reservation was not released: count=%d err=%v", reservedCount, err)
	}

	for index, next := range []string{"contacted", "replied", "meeting"} {
		leadTwo, err = store.TransitionLead(ctx, scope, leadTwo.ID, next, "growth-lead-two-funnel-"+string(rune('a'+index)))
		mustStore(t, err)
	}
	deal, err := store.CreateDeal(ctx, scope, application.DealInput{LeadID: leadTwo.ID, Name: "Test Deal", CustomerID: "Test Customer", Currency: "USD", ValueMinor: 500}, "growth-deal")
	mustStore(t, err)
	conversation, err := store.CreateConversation(ctx, scope, application.ConversationInput{LeadID: leadTwo.ID, DealID: deal.ID, Channel: "email"}, "growth-conversation")
	mustStore(t, err)
	message, err := store.AddConversationMessage(ctx, scope, conversation.ID, application.ConversationMessageInput{Direction: "inbound", Status: "received", Content: map[string]any{"text": "Test reply"}}, "growth-conversation-message")
	mustStore(t, err)
	if message.Direction != "inbound" {
		t.Fatalf("unexpected conversation message: %#v", message)
	}

	quote, err := store.CreateQuote(ctx, scope, application.QuoteInput{
		DealID: deal.ID, CustomerID: deal.CustomerID, Currency: "USD", ValidUntil: time.Now().UTC().Add(24 * time.Hour),
		Items: []application.QuoteItemInput{{SKUVersionID: skuVersion.ID, Quantity: 1, Input: map[string]any{"input": "Test Order Input"}}},
	}, "growth-deal-quote")
	mustStore(t, err)
	if quote.CanonicalDealID != deal.ID {
		t.Fatalf("quote is not bound to canonical deal: %#v", quote)
	}
	deal, err = store.TransitionDeal(ctx, scope, deal.ID, "proposal", "growth-deal-proposal")
	mustStore(t, err)
	quote, err = store.TransitionQuote(ctx, scope, quote.ID, "accepted", "growth-deal-quote-accepted")
	mustStore(t, err)
	deal, err = store.TransitionDeal(ctx, scope, deal.ID, "won", "growth-deal-won")
	mustStore(t, err)
	if deal.Status != "won" || quote.Status != "accepted" {
		t.Fatalf("canonical deal/quote close failed: deal=%#v quote=%#v", deal, quote)
	}

	experiment, err := store.CreateExperiment(ctx, scope, application.ExperimentInput{Name: "Test Experiment", EntityType: "deal", EntityID: deal.ID, Hypothesis: "Test measurable outcome", AllocationBasisPoints: 5000, MetricsDefinition: map[string]any{"conversion": map[string]any{"type": "count"}}}, "growth-experiment")
	mustStore(t, err)
	experiment, err = store.TransitionExperiment(ctx, scope, experiment.ID, application.ExperimentTransitionInput{To: "running"}, "growth-experiment-running")
	mustStore(t, err)
	experiment, err = store.TransitionExperiment(ctx, scope, experiment.ID, application.ExperimentTransitionInput{To: "completed", Result: map[string]any{"outcome": "recorded"}}, "growth-experiment-completed")
	mustStore(t, err)
	if experiment.Status != "completed" {
		t.Fatalf("experiment status=%s", experiment.Status)
	}

	overview, err := store.ListGrowth(ctx, scope)
	mustStore(t, err)
	if len(overview.Segments) != 1 || len(overview.Leads) != 2 || len(overview.ProofInstances) != 1 || len(overview.Campaigns) != 1 || len(overview.Outreach) != 2 || len(overview.Conversations) != 1 || len(overview.Deals) != 1 || len(overview.Experiments) != 1 {
		t.Fatalf("incomplete growth overview: %#v", overview)
	}
	otherOverview, err := store.ListGrowth(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"})
	mustStore(t, err)
	if len(otherOverview.Leads) != 0 || len(otherOverview.Deals) != 0 || len(otherOverview.Outreach) != 0 {
		t.Fatalf("cross-tenant growth visibility: %#v", otherOverview)
	}
	if _, err = store.GetDeal(ctx, tenancy.Scope{TenantID: otherTenantID, ActorID: "other"}, deal.ID); err == nil {
		t.Fatal("cross-tenant deal read succeeded")
	}

	var audits, events int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id=$1 AND object_type IN ('market_segment','icp_definition','lead','lead_evidence','contact','proof_template','proof_request','proof_instance','campaign','campaign_step','suppression_entry','outreach_message','conversation','conversation_message','deal','experiment')`, tenantID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1 AND aggregate_type IN ('market_segment','icp_definition','lead','proof_template','proof_request','campaign','suppression_entry','outreach_message','conversation','deal','experiment')`, tenantID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if audits < 27 || events < 27 {
		t.Fatalf("growth audit/outbox chain incomplete: audits=%d events=%d", audits, events)
	}
}

func cleanupGrowthTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{
		"conversation_messages", "conversations", "outreach_messages", "send_quotas", "suppression_entries",
		"campaign_approvals", "campaign_steps", "campaigns", "proof_instances", "proof_requests", "proof_templates", "experiments",
		"quote_version_items", "quote_versions", "quotes", "deals", "contact_sources", "contacts", "lead_evidence", "leads", "icp_definitions", "market_segments",
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	cleanupTransactionTenant(t, pool, tenantID)
}
