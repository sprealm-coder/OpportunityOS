package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/finance"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/operations"
	orderdomain "github.com/opportunity-os/opportunity-os/services/core-api/internal/order"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
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

func TestAPICORSAllowsWorkspacePortalsOnly(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.opportunity.test,http://127.0.0.1:3010")
	server := New()
	for _, origin := range []string{"http://127.0.0.1:3003", "http://127.0.0.1:3004", "http://127.0.0.1:3005", "https://admin.opportunity.test", "http://127.0.0.1:3010"} {
		request := httptest.NewRequest(http.MethodOptions, "/v1/channels", nil)
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != origin || response.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("portal origin %s was not allowed: status=%d headers=%v", origin, response.Code, response.Header())
		}
	}
	request := httptest.NewRequest(http.MethodOptions, "/v1/channels", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("untrusted origin received CORS access: %v", response.Header())
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

type transactionAPIStore struct {
	application.Store
	application.TransactionStore
	quote orderdomain.Quote
	order orderdomain.Order
}

func (s *transactionAPIStore) CreateQuote(_ context.Context, scope tenancy.Scope, input application.QuoteInput, _ string) (orderdomain.Quote, error) {
	version := orderdomain.QuoteVersion{ID: "quote-version-api", TenantID: scope.TenantID, QuoteID: "quote-api", Version: 1, Currency: input.Currency, AmountMinor: 500, ValidUntil: input.ValidUntil, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC(), Items: []orderdomain.QuoteItem{}}
	s.quote = orderdomain.Quote{ID: "quote-api", TenantID: scope.TenantID, DealID: input.DealID, CustomerID: input.CustomerID, Status: "draft", Version: 1, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Versions: []orderdomain.QuoteVersion{version}}
	return s.quote, nil
}

func (s *transactionAPIStore) ListQuotes(_ context.Context, _ tenancy.Scope) ([]orderdomain.Quote, error) {
	return []orderdomain.Quote{s.quote}, nil
}

func (s *transactionAPIStore) TransitionQuote(_ context.Context, _ tenancy.Scope, _, to, _ string) (orderdomain.Quote, error) {
	s.quote.Status = to
	return s.quote, nil
}

func (s *transactionAPIStore) CreateOrder(_ context.Context, scope tenancy.Scope, quoteVersionID, key string) (orderdomain.Order, error) {
	s.order = orderdomain.Order{ID: "order-api", TenantID: scope.TenantID, CustomerID: s.quote.CustomerID, QuoteVersionID: quoteVersionID, Status: "created", Currency: "USD", AmountMinor: 500, Version: 1, IdempotencyKey: key, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Items: []orderdomain.OrderItem{}}
	return s.order, nil
}

func (s *transactionAPIStore) ListOrders(_ context.Context, _ tenancy.Scope) ([]orderdomain.Order, error) {
	return []orderdomain.Order{s.order}, nil
}

func TestAPITransactionRoutes(t *testing.T) {
	store := &transactionAPIStore{Store: application.NewMemoryStore()}
	server := newHandler(store, true)
	validUntil := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	quote := apiCommand(t, server, http.MethodPost, "/v1/quotes", fmt.Sprintf(`{"deal_id":"Test Deal","customer_id":"Test Customer","currency":"USD","valid_until":%q,"items":[{"sku_version_id":"sku-version-api","quantity":1,"input":{"input":"Test Input"}}]}`, validUntil), "quote-api", http.StatusCreated)
	if quote["status"] != "draft" {
		t.Fatalf("quote status=%v", quote["status"])
	}
	quote = apiCommand(t, server, http.MethodPost, "/v1/quotes/quote-api/transitions", `{"to":"accepted"}`, "quote-accept-api", http.StatusOK)
	if quote["status"] != "accepted" {
		t.Fatalf("accepted quote status=%v", quote["status"])
	}
	createdOrder := apiCommand(t, server, http.MethodPost, "/v1/orders", `{"quote_version_id":"quote-version-api"}`, "order-api", http.StatusCreated)
	if createdOrder["status"] != "created" || createdOrder["quote_version_id"] != "quote-version-api" {
		t.Fatalf("unexpected order response: %#v", createdOrder)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	request.Header.Set("X-Tenant-ID", "tenant-api-flow")
	request.Header.Set("X-Actor-ID", "actor-api-flow")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("order-api")) {
		t.Fatalf("list orders status=%d body=%s", response.Code, response.Body.String())
	}
}

type integrationAPIStore struct {
	application.Store
	application.IntegrationStore
	received application.AdapterIngressRequest
}

func (s *integrationAPIStore) IngestAdapterResult(_ context.Context, request application.AdapterIngressRequest) (operations.AdapterReceipt, error) {
	s.received = request
	now := time.Now().UTC()
	return operations.AdapterReceipt{ID: "receipt-api", TenantID: "tenant-api", AdapterIdentityID: "adapter-api", ExecutionOrderID: "execution-api", WorkflowStepID: "step-api", ExternalEventID: "event-api", Nonce: request.Nonce, ResultStatus: "succeeded", Payload: map[string]any{"ok": true}, ReceivedAt: now, ProcessedAt: &now}, nil
}

func TestAPIAdapterIngressAndExecutionBoundary(t *testing.T) {
	store := &integrationAPIStore{Store: application.NewMemoryStore()}
	server := newHandler(store, true)
	body := []byte(`{"external_event_id":"event-api","execution_id":"execution-api","status":"succeeded"}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/adapter-results", bytes.NewReader(body))
	request.Header.Set("X-Adapter-Key", "adapter-key")
	request.Header.Set("X-Adapter-Timestamp", "123")
	request.Header.Set("X-Adapter-Nonce", "nonce-api")
	request.Header.Set("X-Adapter-Signature", "signature-api")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.received.KeyID != "adapter-key" || store.received.Nonce != "nonce-api" || !bytes.Equal(store.received.Body, body) {
		t.Fatalf("adapter ingress status=%d received=%#v body=%s", response.Code, store.received, response.Body.String())
	}

	transactionStore := &transactionAPIStore{Store: application.NewMemoryStore()}
	server = newHandler(transactionStore, true)
	request = httptest.NewRequest(http.MethodPost, "/v1/executions/execution-api/transitions", bytes.NewBufferString(`{"to":"succeeded","output":{"forged":true}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Tenant-ID", "tenant-api-flow")
	request.Header.Set("X-Actor-ID", "actor-api-flow")
	request.Header.Set("Idempotency-Key", "forged-execution-result")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !bytes.Contains(response.Body.Bytes(), []byte("trusted_adapter_required")) {
		t.Fatalf("browser supplied execution result status=%d body=%s", response.Code, response.Body.String())
	}
}

type financeAPIStore struct {
	application.Store
	application.FinanceStore
	overview finance.Overview
}

func (s *financeAPIStore) CreateWallet(_ context.Context, scope tenancy.Scope, input application.WalletInput, key string) (finance.Wallet, error) {
	wallet := finance.Wallet{ID: "wallet-api", TenantID: scope.TenantID, OwnerType: input.OwnerType, OwnerID: input.OwnerID, Currency: input.Currency, Status: "active", CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	s.overview.Wallets = append(s.overview.Wallets, wallet)
	return wallet, nil
}

func (s *financeAPIStore) ListFinance(_ context.Context, _ tenancy.Scope) (finance.Overview, error) {
	return s.overview, nil
}

func TestAPIFinanceRoutes(t *testing.T) {
	store := &financeAPIStore{Store: application.NewMemoryStore(), overview: finance.Overview{Wallets: []finance.Wallet{}, Accounts: []finance.Account{}, Transactions: []finance.Transaction{}, Holds: []finance.Hold{}, Refunds: []finance.Refund{}, Commissions: []finance.Commission{}, ProviderPayables: []finance.ProviderPayable{}, Settlements: []finance.Settlement{}, ReconciliationRuns: []finance.ReconciliationRun{}}}
	server := newHandler(store, true)
	wallet := apiCommand(t, server, http.MethodPost, "/v1/wallets", `{"owner_type":"customer","owner_id":"Test Customer","currency":"USD"}`, "wallet-api", http.StatusCreated)
	if wallet["id"] != "wallet-api" || wallet["owner_id"] != "Test Customer" {
		t.Fatalf("unexpected wallet response: %#v", wallet)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/finance", nil)
	request.Header.Set("X-Tenant-ID", "tenant-api-flow")
	request.Header.Set("X-Actor-ID", "actor-api-flow")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("wallet-api")) || !bytes.Contains(response.Body.Bytes(), []byte("transactions")) {
		t.Fatalf("finance overview status=%d body=%s", response.Code, response.Body.String())
	}
}
