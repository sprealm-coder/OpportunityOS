package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
)

func TestAPICookieSessionLifecycle(t *testing.T) {
	server := NewWithStore(application.NewMemoryStore())
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/sessions", bytes.NewBufferString(`{"email":"admin@opportunity.local","password":"opportunity-dev"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.SessionCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %#v", cookies)
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodDelete, "/v1/auth/session", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/v1/opportunities", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAPIStrictAuthIgnoresTenantHeadersAndEnforcesRole(t *testing.T) {
	base := application.NewMemoryStore()
	server := NewWithStore(base)
	request := httptest.NewRequest(http.MethodGet, "/v1/opportunities", nil)
	request.Header.Set("X-Tenant-ID", "forged-tenant")
	request.Header.Set("X-Actor-ID", "forged-actor")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("strict auth accepted headers: status=%d", response.Code)
	}

	login := httptest.NewRequest(http.MethodPost, "/v1/auth/sessions", bytes.NewBufferString(`{"email":"admin@opportunity.local","password":"opportunity-dev"}`))
	login.Header.Set("Content-Type", "application/json")
	loggedIn := httptest.NewRecorder()
	server.ServeHTTP(loggedIn, login)
	cookie := loggedIn.Result().Cookies()[0]
	auditorServer := NewWithStore(&sessionRoleStore{Store: base, role: "auditor"})
	request = httptest.NewRequest(http.MethodPost, "/v1/opportunities", bytes.NewBufferString(`{"name":"Forbidden"}`))
	request.AddCookie(cookie)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "forbidden")
	response = httptest.NewRecorder()
	auditorServer.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("auditor create status=%d body=%s", response.Code, response.Body.String())
	}
}

type sessionRoleStore struct {
	application.Store
	role string
}

func (s *sessionRoleStore) ResolveSession(ctx context.Context, token string) (auth.Session, error) {
	session, err := s.Store.ResolveSession(ctx, token)
	session.Role = s.role
	return session, err
}

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
	blueprintID := blueprint["id"].(string)
	for index, next := range []string{"analyzing", "validating", "approved"} {
		blueprint = apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/blueprints/%s/transitions", blueprintID), fmt.Sprintf(`{"to":%q}`, next), fmt.Sprintf("blueprint-%d", index), http.StatusOK)
	}
	capabilityItem := apiCommand(t, server, http.MethodPost, "/v1/capabilities", `{"name":"Test Capability","description":"Neutral capability","definition":{"fixture":true}}`, "capability", http.StatusCreated)
	capabilityID := capabilityItem["id"].(string)
	provider := apiCommand(t, server, http.MethodPost, "/v1/providers", `{"name":"Test Provider"}`, "provider", http.StatusCreated)
	apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/providers/%s/endpoints", provider["id"]), fmt.Sprintf(`{"capability_id":%q,"adapter_type":"mock_realtime","adapter_version":"v1"}`, capabilityID), "endpoint", http.StatusCreated)
	product := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/blueprints/%s/products", blueprintID), `{"name":"Test Product"}`, "product", http.StatusCreated)
	productID := product["id"].(string)
	versionBody := fmt.Sprintf(`{"input_schema":{"type":"object","properties":{"input":{"type":"string"}}},"output_schema":{"type":"object","properties":{"result":{"type":"string"},"units":{"type":"integer"}}},"form_schema":{"type":"object","properties":{"input":{"type":"string"}}},"capability_ids":[%q],"workflow":{"name":"Test Workflow","version":1,"nodes":[{"id":"start","type":"start"},{"id":"execute","type":"realtime_call"},{"id":"meter","type":"meter"},{"id":"end","type":"end"}],"edges":[{"from":"start","to":"execute"},{"from":"execute","to":"meter"},{"from":"meter","to":"end"}]},"metering":{"name":"Test Meter","unit":"test_unit","field":"units","version":1},"price_book":{"currency":"USD","version":1,"rules":[{"kind":"flat","flat_minor":500}]},"route_policy":{"name":"Test Route","strategy":"priority","version":1},"delivery_mode":"workflow","compliance_profile":{"classification":"test"},"growth_playbook":{"fixture":true}}`, capabilityID)
	productVersion := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/products/%s/versions", productID), versionBody, "product-version", http.StatusCreated)
	productVersionID := productVersion["id"].(string)
	sku := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/products/%s/skus", productID), `{"code":"TEST-SKU","name":"Test SKU"}`, "sku", http.StatusCreated)
	apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/skus/%s/versions", sku["id"]), fmt.Sprintf(`{"product_version_id":%q,"entitlements":{"test_limit":10}}`, productVersionID), "sku-version", http.StatusCreated)
	publication := apiCommand(t, server, http.MethodPost, fmt.Sprintf("/v1/products/%s/publications", productID), fmt.Sprintf(`{"product_version_id":%q}`, productVersionID), "publication", http.StatusCreated)
	if publication["status"] != "published" {
		t.Fatalf("publication status=%v", publication["status"])
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
