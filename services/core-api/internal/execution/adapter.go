package execution

import (
	"context"
	"fmt"
	"sync"
)

type Request struct {
	RequestID, IdempotencyKey, TenantID, BrandID, OrderID, ProductVersionID, SKUVersionID, WorkflowVersionID, ProviderEndpointID, AdapterVersion, PricingVersionID, RoutingVersionID, TraceID string
	Input                                                                                                                                                                                     map[string]any
}

func (r Request) Validate() error {
	required := map[string]string{"request_id": r.RequestID, "idempotency_key": r.IdempotencyKey, "tenant_id": r.TenantID, "brand_id": r.BrandID, "order_id": r.OrderID, "product_version_id": r.ProductVersionID, "sku_version_id": r.SKUVersionID, "workflow_version_id": r.WorkflowVersionID, "provider_endpoint_id": r.ProviderEndpointID, "adapter_version": r.AdapterVersion, "pricing_version_id": r.PricingVersionID, "routing_version_id": r.RoutingVersionID, "trace_id": r.TraceID}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

type Estimate struct {
	CostMinor int64
	Currency  string
	Capacity  bool
}
type Result struct {
	ExternalID, Status  string
	Output, Usage, Cost map[string]any
}
type Health struct {
	Healthy bool
	Message string
}

type Adapter interface {
	Validate(context.Context, Request) error
	Estimate(context.Context, Request) (Estimate, error)
	Reserve(context.Context, Request) error
	Execute(context.Context, Request) (Result, error)
	Poll(context.Context, Request, string) (Result, error)
	Cancel(context.Context, Request, string) error
	NormalizeResult(Result) (map[string]any, error)
	NormalizeUsage(Result) (map[string]any, error)
	NormalizeCost(Result) (map[string]any, error)
	NormalizeError(error) error
	Reconcile(context.Context, Request, string) (Result, error)
	Compensate(context.Context, Request, Result) error
	Health(context.Context) (Health, error)
	Capabilities() []string
}

type MockAdapter struct {
	Kind    string
	mu      sync.Mutex
	results map[string]Result
}

func NewMock(kind string) *MockAdapter                             { return &MockAdapter{Kind: kind, results: map[string]Result{}} }
func (m *MockAdapter) Validate(_ context.Context, r Request) error { return r.Validate() }
func (m *MockAdapter) Estimate(_ context.Context, r Request) (Estimate, error) {
	if err := r.Validate(); err != nil {
		return Estimate{}, err
	}
	return Estimate{CostMinor: 100, Currency: "USD", Capacity: true}, nil
}
func (m *MockAdapter) Reserve(_ context.Context, r Request) error { return r.Validate() }
func (m *MockAdapter) Execute(_ context.Context, r Request) (Result, error) {
	if err := r.Validate(); err != nil {
		return Result{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if result, ok := m.results[r.TenantID+"/"+r.IdempotencyKey]; ok {
		return result, nil
	}
	result := Result{ExternalID: "mock-" + r.RequestID, Status: "succeeded", Output: map[string]any{"artifact": "Test Proof Artifact"}, Usage: map[string]any{"units": int64(2)}, Cost: map[string]any{"minor": int64(100), "currency": "USD"}}
	m.results[r.TenantID+"/"+r.IdempotencyKey] = result
	return result, nil
}
func (m *MockAdapter) Poll(_ context.Context, _ Request, id string) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, result := range m.results {
		if result.ExternalID == id {
			return result, nil
		}
	}
	return Result{}, fmt.Errorf("result not found")
}
func (m *MockAdapter) Cancel(_ context.Context, _ Request, _ string) error { return nil }
func (m *MockAdapter) NormalizeResult(r Result) (map[string]any, error)    { return r.Output, nil }
func (m *MockAdapter) NormalizeUsage(r Result) (map[string]any, error)     { return r.Usage, nil }
func (m *MockAdapter) NormalizeCost(r Result) (map[string]any, error)      { return r.Cost, nil }
func (m *MockAdapter) NormalizeError(err error) error                      { return err }
func (m *MockAdapter) Reconcile(ctx context.Context, r Request, id string) (Result, error) {
	return m.Poll(ctx, r, id)
}
func (m *MockAdapter) Compensate(_ context.Context, _ Request, _ Result) error { return nil }
func (m *MockAdapter) Health(_ context.Context) (Health, error) {
	return Health{Healthy: true, Message: "mock adapter ready"}, nil
}
func (m *MockAdapter) Capabilities() []string { return []string{m.Kind} }

type RealtimeAdapter interface{ Adapter }
type AsyncTaskAdapter interface{ Adapter }
type ProvisioningAdapter interface{ Adapter }
type ManualServiceAdapter interface{ Adapter }
