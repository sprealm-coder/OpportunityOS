package openapi

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPICommercialPlatformContract(t *testing.T) {
	contents, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		OpenAPI string `yaml:"openapi"`
		Info    struct {
			Version string `yaml:"version"`
		} `yaml:"info"`
		Paths map[string]any `yaml:"paths"`
	}
	if err = yaml.Unmarshal(contents, &document); err != nil {
		t.Fatal(err)
	}
	if document.OpenAPI != "3.1.0" || document.Info.Version != "0.8.0" {
		t.Fatalf("unexpected OpenAPI version: %s / %s", document.OpenAPI, document.Info.Version)
	}
	required := []string{
		"/v1/quotes", "/v1/quotes/{id}", "/v1/quotes/{id}/transitions",
		"/v1/orders", "/v1/orders/{id}", "/v1/orders/{id}/transitions",
		"/v1/executions/{id}/transitions", "/v1/deliveries/{id}/transitions",
		"/v1/executions/{id}/usage", "/v1/executions/{id}/provider-costs",
		"/v1/executions/{id}/customer-charges",
		"/v1/finance", "/v1/wallets", "/v1/wallets/{id}/adjustments",
		"/v1/orders/{id}/holds", "/v1/holds/{id}/releases",
		"/v1/customer-charges/{id}/postings", "/v1/customer-charges/{id}/refunds",
		"/v1/customer-charges/{id}/commissions", "/v1/provider-costs/{id}/payables",
		"/v1/settlements", "/v1/reconciliation-runs",
		"/v1/growth", "/v1/market-segments", "/v1/market-segments/{id}/icps",
		"/v1/leads", "/v1/leads/{id}/evidence", "/v1/leads/{id}/transitions", "/v1/leads/{id}/contacts",
		"/v1/proof-templates", "/v1/leads/{id}/proof-requests", "/v1/proof-requests/{id}/generate", "/v1/proof-requests/{id}/reviews",
		"/v1/campaigns", "/v1/campaigns/{id}/steps", "/v1/campaigns/{id}/transitions", "/v1/campaigns/{id}/reviews",
		"/v1/suppressions", "/v1/campaigns/{id}/outreach", "/v1/outreach/{id}/transitions",
		"/v1/conversations", "/v1/conversations/{id}/messages", "/v1/deals", "/v1/deals/{id}",
		"/v1/deals/{id}/transitions", "/v1/deals/{id}/quotes", "/v1/experiments", "/v1/experiments/{id}/transitions",
		"/v1/channels", "/v1/reseller-levels", "/v1/resellers", "/v1/attribution-rules",
		"/v1/lead-ownerships", "/v1/customer-ownerships", "/v1/ownership-transfers", "/v1/ownership-transfers/{id}/reviews",
		"/v1/commission-rules", "/v1/commission-locks", "/v1/settlement-cycles",
		"/v1/suppliers", "/v1/suppliers/{id}/capabilities", "/v1/providers/{id}/supplier-binding",
		"/v1/supplier-contracts", "/v1/supplier-contracts/{id}/transitions", "/v1/supplier-contracts/{id}/reviews",
		"/v1/supplier-rates", "/v1/supplier-quality-records", "/v1/developers", "/v1/publishers", "/v1/listings",
		"/v1/listings/{id}/versions", "/v1/listings/{id}/transitions", "/v1/listings/{id}/reviews", "/v1/sandbox-runs", "/v1/quality-scores",
		"/v1/marketplace-disputes", "/v1/marketplace-disputes/{id}/resolutions", "/v1/takedowns", "/v1/takedowns/{id}/reviews",
		"/v1/intelligence", "/v1/sources", "/v1/sources/{id}/signals", "/v1/signals/{id}/promotions",
		"/v1/analytics/outcomes", "/v1/outcome-feedback", "/v1/adapter-identities", "/v1/executions/{id}/workflow-runs",
		"/v1/workflow-steps/leases", "/v1/adapter-results", "/v1/operations", "/v1/outbox/{id}/replays", "/v1/operational-alerts/{id}/acknowledgements",
	}
	for _, path := range required {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("missing platform path %s", path)
		}
	}
}
