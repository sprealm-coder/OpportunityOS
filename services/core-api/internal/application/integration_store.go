package application

import (
	"context"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/analytics"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/intelligence"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/operations"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type SourceInput struct {
	Name          string         `json:"name"`
	ConnectorType string         `json:"connector_type"`
	Config        map[string]any `json:"config"`
}

type SignalInput struct {
	ExternalID  string         `json:"external_id"`
	Fingerprint string         `json:"fingerprint"`
	Payload     map[string]any `json:"payload"`
	Normalized  map[string]any `json:"normalized"`
	OccurredAt  time.Time      `json:"occurred_at"`
}

type SignalPromotionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
	Confidence  int    `json:"confidence"`
}

type OutcomeFeedbackInput struct {
	OpportunityID    string         `json:"opportunity_id"`
	OrderID          string         `json:"order_id"`
	ExecutionOrderID string         `json:"execution_order_id"`
	Metrics          map[string]any `json:"metrics"`
	Evidence         map[string]any `json:"evidence"`
}

type AdapterIdentityInput struct {
	Name               string `json:"name"`
	KeyID              string `json:"key_id"`
	ProviderEndpointID string `json:"provider_endpoint_id"`
	SecretRef          string `json:"secret_ref"`
}

type WorkflowRunInput struct {
	MaxAttempts int `json:"max_attempts"`
}

type WorkflowLeaseInput struct {
	AdapterIdentityID string `json:"adapter_identity_id"`
	LeaseSeconds      int    `json:"lease_seconds"`
}

type AdapterIngressRequest struct {
	KeyID     string
	Timestamp string
	Nonce     string
	Signature string
	Body      []byte
}

type IntegrationStore interface {
	ListIntelligence(context.Context, tenancy.Scope) (intelligence.Overview, error)
	CreateSource(context.Context, tenancy.Scope, SourceInput, string) (intelligence.Source, error)
	ImportSignal(context.Context, tenancy.Scope, string, SignalInput, string) (intelligence.Signal, error)
	PromoteSignal(context.Context, tenancy.Scope, string, SignalPromotionInput, string) (opportunity.Opportunity, error)

	ListAnalytics(context.Context, tenancy.Scope) (analytics.Overview, error)
	RecordOutcomeFeedback(context.Context, tenancy.Scope, OutcomeFeedbackInput, string) (analytics.OutcomeFeedback, error)

	RegisterAdapterIdentity(context.Context, tenancy.Scope, AdapterIdentityInput, string) (operations.AdapterIdentity, error)
	StartWorkflowRun(context.Context, tenancy.Scope, string, WorkflowRunInput, string) (operations.WorkflowRun, error)
	LeaseWorkflowStep(context.Context, tenancy.Scope, WorkflowLeaseInput, string) (operations.WorkflowStep, error)
	IngestAdapterResult(context.Context, AdapterIngressRequest) (operations.AdapterReceipt, error)

	ListOperations(context.Context, tenancy.Scope) (operations.Overview, error)
	ReplayOutbox(context.Context, tenancy.Scope, string, string, string) (operations.OutboxReplay, error)
	AcknowledgeOperationalAlert(context.Context, tenancy.Scope, string, string) (operations.Alert, error)
}
