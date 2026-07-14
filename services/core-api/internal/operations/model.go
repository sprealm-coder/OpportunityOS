package operations

import "time"

type WorkflowRun struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	DefinitionID     string         `json:"definition_id"`
	ExecutionOrderID string         `json:"execution_order_id"`
	IdempotencyKey   string         `json:"idempotency_key"`
	Status           string         `json:"status"`
	Variables        map[string]any `json:"variables"`
	CreatedBy        string         `json:"created_by"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Steps            []WorkflowStep `json:"steps"`
}

type WorkflowStep struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	RunID            string         `json:"run_id"`
	ExecutionOrderID string         `json:"execution_order_id"`
	NodeID           string         `json:"node_id"`
	NodeType         string         `json:"node_type"`
	Status           string         `json:"status"`
	Attempt          int            `json:"attempt"`
	MaxAttempts      int            `json:"max_attempts"`
	LockedBy         string         `json:"locked_by,omitempty"`
	LockedUntil      *time.Time     `json:"locked_until,omitempty"`
	NextRetryAt      *time.Time     `json:"next_retry_at,omitempty"`
	Output           map[string]any `json:"output"`
	Error            map[string]any `json:"error"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
}

type AdapterIdentity struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	Name               string    `json:"name"`
	KeyID              string    `json:"key_id"`
	ProviderEndpointID string    `json:"provider_endpoint_id"`
	SecretRef          string    `json:"secret_ref"`
	Status             string    `json:"status"`
	CreatedBy          string    `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type AdapterResultInput struct {
	ExternalEventID string         `json:"external_event_id"`
	ExecutionID     string         `json:"execution_id"`
	Status          string         `json:"status"`
	ExternalID      string         `json:"external_id,omitempty"`
	Output          map[string]any `json:"output,omitempty"`
	Error           map[string]any `json:"error,omitempty"`
}

type AdapterReceipt struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	AdapterIdentityID string         `json:"adapter_identity_id"`
	ExecutionOrderID  string         `json:"execution_order_id"`
	WorkflowStepID    string         `json:"workflow_step_id"`
	ExternalEventID   string         `json:"external_event_id"`
	Nonce             string         `json:"nonce"`
	ResultStatus      string         `json:"result_status"`
	Payload           map[string]any `json:"payload"`
	ReceivedAt        time.Time      `json:"received_at"`
	ProcessedAt       *time.Time     `json:"processed_at,omitempty"`
}

type OutboxHealth struct {
	Pending          int        `json:"pending"`
	RetryScheduled   int        `json:"retry_scheduled"`
	DeadLetter       int        `json:"dead_letter"`
	OldestPendingAt  *time.Time `json:"oldest_pending_at,omitempty"`
	LastPublishedAt  *time.Time `json:"last_published_at,omitempty"`
	OldestPendingAge int64      `json:"oldest_pending_age_seconds"`
}

type Alert struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	AlertType      string         `json:"alert_type"`
	Severity       string         `json:"severity"`
	Status         string         `json:"status"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	Message        string         `json:"message"`
	Details        map[string]any `json:"details"`
	CreatedAt      time.Time      `json:"created_at"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
}

type OutboxReplay struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	OutboxEventID      string    `json:"outbox_event_id"`
	Reason             string    `json:"reason"`
	PreviousRetryCount int       `json:"previous_retry_count"`
	RequestedBy        string    `json:"requested_by"`
	RequestedAt        time.Time `json:"requested_at"`
}

type DeploymentCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Overview struct {
	Outbox            OutboxHealth      `json:"outbox"`
	WorkflowRuns      []WorkflowRun     `json:"workflow_runs"`
	AdapterIdentities []AdapterIdentity `json:"adapter_identities"`
	AdapterReceipts   []AdapterReceipt  `json:"adapter_receipts"`
	Alerts            []Alert           `json:"alerts"`
	DeploymentChecks  []DeploymentCheck `json:"deployment_checks"`
}
