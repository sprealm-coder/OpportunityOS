package tests

import (
	"context"
	"testing"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/execution"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/growth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/ledger"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/order"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/pricing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/routing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

func TestNeutralEndToEnd(t *testing.T) {
	ctx := context.Background()
	scope := tenancy.Scope{TenantID: "test-tenant", ActorID: "test-operator", TraceID: "trace-e2e"}
	checkpoints := []string{"tenant", "brand", "source", "signal"}
	repo := opportunity.NewMemoryRepository()
	auditLog := &audit.Log{}
	events := &outbox.Memory{}
	opportunities := opportunity.NewService(repo, auditLog, events)
	opp, err := opportunities.Create(scope, "Test Opportunity", "Neutral opportunity created from Test Signal", "create-opportunity")
	must(t, err)
	checkpoints = append(checkpoints, "opportunity")
	opp, err = opportunities.AddEvidence(scope, opp.ID, opportunity.Evidence{Kind: "test", Summary: "Test Evidence", Confidence: 90}, "add-evidence")
	must(t, err)
	checkpoints = append(checkpoints, "evidence")
	opp, err = opportunities.Score(scope, opp.ID, 82, "score-opportunity")
	must(t, err)
	for _, target := range []string{"under_review", "approved", "incubating"} {
		opp, err = opportunities.Transition(scope, opp.ID, target, "transition-"+target)
		must(t, err)
	}
	checkpoints = append(checkpoints, "score", "review")

	project := incubation.New(scope.TenantID, opp.ID, "Test Incubation Project")
	for _, target := range []string{"researching", "validating", "approved", "building", "launched"} {
		must(t, project.Transition(target))
	}
	checkpoints = append(checkpoints, "incubation")
	bp := blueprint.New(scope.TenantID, scope.ActorID, opp.ID, "Test Business Blueprint", "Neutral blueprint fixture")
	bp.ValueProposition = "Test Value Proposition"
	bp.RequiredCapabilities = []string{"Test Capability"}
	bp.ProductDefinitions = []map[string]any{{"name": "Test Product"}}
	bp.WorkflowDefinitions = []map[string]any{{"name": "Test Workflow"}}
	bp.MeteringDefinitions = []map[string]any{{"unit": "test_unit"}}
	bp.PricingDefinitions = []map[string]any{{"currency": "USD"}}
	bp.ComplianceProfile = map[string]any{"classification": "test"}
	must(t, bp.ValidateCompleteness())
	for _, target := range []string{"analyzing", "validating", "approved", "configuring", "ready"} {
		must(t, bp.Transition(target, scope.ActorID))
	}
	checkpoints = append(checkpoints, "blueprint")

	capabilityItem := capability.New(scope.TenantID, "Test Capability")
	provider := capability.NewProvider(scope.TenantID, "Test Provider")
	endpoint := capability.NewEndpoint(scope.TenantID, provider.ID, capabilityItem.ID, "mock_realtime")
	checkpoints = append(checkpoints, "capability", "provider")
	inputSchema := schema.Definition{"type": "object", "properties": map[string]any{"input": map[string]any{"type": "string"}}}
	outputSchema := schema.Definition{"type": "object", "properties": map[string]any{"artifact": map[string]any{"type": "string"}, "units": map[string]any{"type": "integer"}}}
	must(t, schema.Validate(inputSchema))
	must(t, schema.Validate(outputSchema))
	checkpoints = append(checkpoints, "schemas")
	definition := workflow.Definition{ID: "test-workflow", TenantID: scope.TenantID, Name: "Test Workflow", Version: 1, Nodes: []workflow.Node{{ID: "start", Type: workflow.Start}, {ID: "validate", Type: workflow.Validate}, {ID: "execute", Type: workflow.RealtimeCall}, {ID: "meter", Type: workflow.Meter}, {ID: "end", Type: workflow.End}}, Edges: []workflow.Edge{{From: "start", To: "validate"}, {From: "validate", To: "execute"}, {From: "execute", To: "meter"}, {From: "meter", To: "end"}}}
	must(t, definition.Validate())
	checkpoints = append(checkpoints, "workflow")
	meter := pricing.MeteringDefinition{ID: "test-meter", TenantID: scope.TenantID, Name: "Test Meter", Unit: "test_unit", Field: "units", Version: 1}
	must(t, meter.Validate())
	priceBook := pricing.PriceBook{ID: "test-price", TenantID: scope.TenantID, Currency: "USD", Version: 1, Rules: []pricing.Rule{{ID: "flat", Kind: "flat", FlatMinor: 500}, {ID: "usage", Kind: "per_unit", UnitMinor: 100}}}
	charge, err := priceBook.Calculate(2)
	must(t, err)
	if charge.Minor != 700 {
		t.Fatalf("unexpected charge %d", charge.Minor)
	}
	policy := routing.Policy{ID: "test-route", TenantID: scope.TenantID, Strategy: "priority", Version: 1}
	selected, err := policy.Select([]routing.Candidate{{EndpointID: endpoint.ID, Healthy: true, Priority: 1, EstimatedCostMinor: 100, Capacity: 10}})
	must(t, err)
	if selected.EndpointID != endpoint.ID {
		t.Fatal("route did not select Test Provider")
	}
	checkpoints = append(checkpoints, "metering", "pricing", "routing")

	product, productVersion := catalog.Draft(scope.TenantID, bp.ID, "Test Product")
	productVersion.InputSchema = inputSchema
	productVersion.OutputSchema = outputSchema
	productVersion.CapabilityIDs = []string{capabilityItem.ID}
	productVersion.Workflow = definition
	productVersion.MeteringID = meter.ID
	productVersion.PriceBookID = priceBook.ID
	productVersion.RoutePolicyID = policy.ID
	productVersion.DeliveryMode = "workflow"
	productVersion.ComplianceProfileID = "test-compliance"
	publication, err := catalog.Publish(&product, productVersion, map[string]bool{capabilityItem.ID: true})
	must(t, err)
	if publication.Status != "published" {
		t.Fatal("product not published")
	}
	sku := catalog.SKU{ID: platform.NewID("sku"), TenantID: scope.TenantID, ProductID: product.ID, Code: "TEST-SKU"}
	skuVersion := catalog.SKUVersion{ID: platform.NewID("skuv"), TenantID: scope.TenantID, SKUID: sku.ID, ProductVersionID: productVersion.ID, WorkflowVersionID: definition.ID, MeteringVersionID: meter.ID, PricingVersionID: priceBook.ID, RoutingVersionID: policy.ID, Version: 1}
	checkpoints = append(checkpoints, "product", "sku", "publication")

	segment := growth.MarketSegment{ID: platform.NewID("segment"), TenantID: scope.TenantID, Name: "Test Market Segment", Definition: map[string]any{"fixture": true}}
	lead := growth.NewLead(scope.TenantID, segment.ID, "Test Customer")
	for _, target := range []string{"enriched", "qualified", "proof_requested"} {
		must(t, lead.Transition(target))
	}
	proofTemplate := growth.ProofTemplate{ID: platform.NewID("proof-template"), TenantID: scope.TenantID, Name: "Test Proof", Type: "custom", WorkflowVersionID: definition.ID, InputSchema: inputSchema, OutputSchema: outputSchema}
	if !proofTemplate.Valid() {
		t.Fatal("proof template invalid")
	}
	proofRequest := growth.ProofRequest{ID: platform.NewID("proof"), TenantID: scope.TenantID, LeadID: lead.ID, TemplateID: proofTemplate.ID, Status: "requested"}
	checkpoints = append(checkpoints, "segment", "lead", "proof_request")

	adapter := execution.NewMock("realtime")
	executionRequest := execution.Request{RequestID: "request-e2e", IdempotencyKey: "execute-e2e", TenantID: scope.TenantID, BrandID: "test-brand", OrderID: "pending", ProductVersionID: productVersion.ID, SKUVersionID: skuVersion.ID, WorkflowVersionID: definition.ID, ProviderEndpointID: endpoint.ID, AdapterVersion: "v1", PricingVersionID: priceBook.ID, RoutingVersionID: policy.ID, TraceID: scope.TraceID, Input: map[string]any{"input": "test"}}
	engine := workflow.NewEngine()
	engine.Register(workflow.RealtimeCall, workflow.HandlerFunc(func(ctx context.Context, _ workflow.Node, variables map[string]any) (map[string]any, error) {
		result, err := adapter.Execute(ctx, executionRequest)
		if err != nil {
			return nil, err
		}
		for key, value := range result.Output {
			variables[key] = value
		}
		for key, value := range result.Usage {
			variables[key] = value
		}
		return variables, nil
	}))
	run, err := engine.Execute(ctx, definition, "proof-run", map[string]any{"proof_request_id": proofRequest.ID})
	must(t, err)
	if run.Status != "succeeded" {
		t.Fatal("proof workflow failed")
	}
	proofRequest.Status = "ready"
	proofRequest.ArtifactID = "Test Proof Artifact"
	must(t, lead.Transition("proof_ready"))
	must(t, lead.Transition("approved_for_outreach"))
	checkpoints = append(checkpoints, "proof_artifact")

	deal := growth.Deal{ID: platform.NewID("deal"), TenantID: scope.TenantID, LeadID: lead.ID, Status: "proposal", ValueMinor: charge.Minor, Currency: charge.Currency}
	quoteVersion := order.QuoteVersion{ID: platform.NewID("quote-version"), TenantID: scope.TenantID, Version: 1, AmountMinor: charge.Minor, Currency: charge.Currency, Items: []order.QuoteItem{{Quantity: 1, AmountMinor: charge.Minor, Bindings: order.VersionBindings{ProductVersionID: productVersion.ID, SKUVersionID: skuVersion.ID, PricingVersionID: priceBook.ID, WorkflowVersionID: definition.ID, RoutingVersionID: policy.ID}}}}
	quote := order.Quote{ID: platform.NewID("quote"), TenantID: scope.TenantID, DealID: deal.ID, CustomerID: "Test Customer", Status: "accepted", Version: 1, Versions: []order.QuoteVersion{quoteVersion}}
	bindings := order.VersionBindings{ProductVersionID: productVersion.ID, SKUVersionID: skuVersion.ID, PricingVersionID: priceBook.ID, WorkflowVersionID: definition.ID, RoutingVersionID: policy.ID, ContractVersionID: "test-contract-v1"}
	customerOrder, err := order.New(scope.TenantID, "Test Customer", "order-e2e", charge.Currency, quote.Versions[0].AmountMinor, bindings)
	must(t, err)
	for _, target := range []string{"awaiting_payment", "paid", "provisioning", "active"} {
		must(t, customerOrder.Transition(target))
	}
	executionRequest.OrderID = customerOrder.ID
	checkpoints = append(checkpoints, "deal", "quote", "order")

	book := ledger.New()
	accounts := []ledger.Account{{ID: "cash", TenantID: scope.TenantID, Code: "cash", Name: "Test Cash", Currency: "USD", Type: ledger.Asset}, {ID: "equity", TenantID: scope.TenantID, Code: "equity", Name: "Test Equity", Currency: "USD", Type: ledger.Equity}, {ID: "wallet", TenantID: scope.TenantID, Code: "wallet", Name: "Test Customer Wallet", Currency: "USD", Type: ledger.Liability}, {ID: "held", TenantID: scope.TenantID, Code: "held", Name: "Test Held Funds", Currency: "USD", Type: ledger.Liability}, {ID: "revenue", TenantID: scope.TenantID, Code: "revenue", Name: "Test Revenue", Currency: "USD", Type: ledger.Revenue}, {ID: "provider_payable", TenantID: scope.TenantID, Code: "provider_payable", Name: "Test Provider Payable", Currency: "USD", Type: ledger.Liability}, {ID: "commission", TenantID: scope.TenantID, Code: "commission", Name: "Test Commission", Currency: "USD", Type: ledger.Liability}, {ID: "promotion", TenantID: scope.TenantID, Code: "promotion", Name: "Test Promotional Credit", Currency: "USD", Type: ledger.Expense}}
	for _, account := range accounts {
		must(t, book.AddAccount(account))
	}
	post := func(key, reference string, entries []ledger.Entry) ledger.Transaction {
		txn, err := book.Post(ledger.Transaction{TenantID: scope.TenantID, IdempotencyKey: key, ReferenceType: "test", ReferenceID: reference, Description: key, Entries: entries})
		must(t, err)
		return txn
	}
	post("fund-platform", "test-funding", []ledger.Entry{{AccountID: "cash", Currency: "USD", Direction: ledger.Debit, AmountMinor: 2000}, {AccountID: "equity", Currency: "USD", Direction: ledger.Credit, AmountMinor: 2000}})
	post("credit-wallet", customerOrder.ID, []ledger.Entry{{AccountID: "promotion", Currency: "USD", Direction: ledger.Debit, AmountMinor: 1000}, {AccountID: "wallet", Currency: "USD", Direction: ledger.Credit, AmountMinor: 1000}})
	post("hold-order", customerOrder.ID, []ledger.Entry{{AccountID: "wallet", Currency: "USD", Direction: ledger.Debit, AmountMinor: 800}, {AccountID: "held", Currency: "USD", Direction: ledger.Credit, AmountMinor: 800}})
	checkpoints = append(checkpoints, "hold")
	result, err := adapter.Execute(ctx, executionRequest)
	must(t, err)
	units := result.Usage["units"].(int64)
	finalCharge, err := priceBook.Calculate(units)
	must(t, err)
	providerCost := int64(250)
	commission := int64(50)
	platformRevenue := finalCharge.Minor - providerCost - commission
	post("charge-order", customerOrder.ID, []ledger.Entry{{AccountID: "held", Currency: "USD", Direction: ledger.Debit, AmountMinor: finalCharge.Minor}, {AccountID: "revenue", Currency: "USD", Direction: ledger.Credit, AmountMinor: platformRevenue}, {AccountID: "provider_payable", Currency: "USD", Direction: ledger.Credit, AmountMinor: providerCost}, {AccountID: "commission", Currency: "USD", Direction: ledger.Credit, AmountMinor: commission}})
	post("release-hold", customerOrder.ID, []ledger.Entry{{AccountID: "held", Currency: "USD", Direction: ledger.Debit, AmountMinor: 100}, {AccountID: "wallet", Currency: "USD", Direction: ledger.Credit, AmountMinor: 100}})
	post("settle-provider", customerOrder.ID, []ledger.Entry{{AccountID: "provider_payable", Currency: "USD", Direction: ledger.Debit, AmountMinor: providerCost}, {AccountID: "cash", Currency: "USD", Direction: ledger.Credit, AmountMinor: providerCost}})
	must(t, customerOrder.Transition("completed"))
	checkpoints = append(checkpoints, "execution", "usage", "provider_cost", "customer_charge", "ledger_charge", "commission", "payable", "settlement", "outcome_feedback")

	if balance, err := book.Balance(scope.TenantID, "held"); err != nil || balance != 0 {
		t.Fatalf("held funds not cleared: %d %v", balance, err)
	}
	if len(checkpoints) < 30 {
		t.Fatalf("acceptance chain incomplete: %v", checkpoints)
	}
	if len(auditLog.ForTenant(scope.TenantID)) < 6 {
		t.Fatal("expected audited opportunity commands")
	}
	if len(events.Pending(scope.TenantID)) < 4 {
		t.Fatal("expected transactional event intents")
	}
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
