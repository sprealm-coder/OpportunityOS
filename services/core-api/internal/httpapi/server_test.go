package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIRequiresTenantAndIdempotency(t *testing.T) {
	server := New()
	request := httptest.NewRequest(http.MethodGet, "/v1/opportunities", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", response.Code)
	}
	body := []byte(`{"name":"Test Opportunity","description":"Neutral"}`)
	request = httptest.NewRequest(http.MethodPost, "/v1/opportunities", bytes.NewReader(body))
	request.Header.Set("X-Tenant-ID", "tenant-a")
	request.Header.Set("X-Actor-ID", "actor-a")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected idempotency error, got %d", response.Code)
	}
}

func TestAPIControlPlaneFlow(t *testing.T) {
	server := New()
	created := apiCommand(t, server, http.MethodPost, "/v1/opportunities", `{"name":"API Flow","description":"Persistent control flow"}`, "create", http.StatusCreated)
	id := created["id"].(string)
	item := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/evidence", id), `{"kind":"interview","summary":"Demand confirmed","confidence":90}`, "evidence", http.StatusOK)
	if item["status"] != "enriched" {
		t.Fatalf("evidence status=%v", item["status"])
	}
	item = apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/score", id), `{"score":84}`, "score", http.StatusOK)
	item = apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/transitions", id), `{"to":"under_review"}`, "review-start", http.StatusOK)
	item = apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/reviews", id), `{"decision":"approved","rationale":"Gate passed"}`, "review", http.StatusOK)
	if item["status"] != "approved" {
		t.Fatalf("review status=%v", item["status"])
	}
	project := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/incubations", id), `{"name":"API Incubation"}`, "incubation", http.StatusCreated)
	if project["status"] != "draft" {
		t.Fatalf("incubation status=%v", project["status"])
	}
	blueprint := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/opportunities/%s/blueprints", id), `{"name":"API Blueprint","value_proposition":"Verified value","required_capabilities":["Test Capability"],"product_definitions":[{"name":"Test Product"}],"workflow_definitions":[{"name":"Test Workflow"}],"metering_definitions":[{"unit":"test_unit"}],"pricing_definitions":[{"currency":"USD"}],"compliance_profile":{"classification":"test"}}`, "blueprint", http.StatusCreated)
	if blueprint["status"] != "draft" {
		t.Fatalf("blueprint status=%v", blueprint["status"])
	}
}

func apiCommand(t *testing.T, server http.Handler, method, path, body, key string, expectedStatus int) map[string]any {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Tenant-ID", "tenant-api-flow")
	request.Header.Set("X-Actor-ID", "actor-api-flow")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestAPIOpportunityFlowAndSecurityHeaders(t *testing.T) {
	server := New()
	body := []byte(`{"name":"Test Opportunity","description":"Neutral hypothesis"}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/opportunities", bytes.NewReader(body))
	request.Header.Set("X-Tenant-ID", "tenant-a")
	request.Header.Set("X-Actor-ID", "actor-a")
	request.Header.Set("Idempotency-Key", "create-1")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security headers")
	}
	var created map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created["tenant_id"] != "tenant-a" {
		t.Fatalf("unexpected tenant: %#v", created)
	}
}
