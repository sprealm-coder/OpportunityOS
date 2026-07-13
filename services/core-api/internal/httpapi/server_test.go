package httpapi

import (
	"bytes"
	"encoding/json"
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
