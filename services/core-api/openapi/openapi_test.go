package openapi

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIFinanceContract(t *testing.T) {
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
	if document.OpenAPI != "3.1.0" || document.Info.Version != "0.5.0" {
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
	}
	for _, path := range required {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("missing transaction path %s", path)
		}
	}
}
