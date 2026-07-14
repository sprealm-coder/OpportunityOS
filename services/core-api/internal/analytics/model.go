package analytics

import "time"

type OutcomeFeedback struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	OpportunityID    string         `json:"opportunity_id"`
	OrderID          string         `json:"order_id"`
	ExecutionOrderID string         `json:"execution_order_id,omitempty"`
	Status           string         `json:"status"`
	Metrics          map[string]any `json:"metrics"`
	Evidence         map[string]any `json:"evidence"`
	IdempotencyKey   string         `json:"idempotency_key"`
	Version          int            `json:"version"`
	CreatedBy        string         `json:"created_by"`
	ValidatedAt      time.Time      `json:"validated_at"`
	CreatedAt        time.Time      `json:"created_at"`
}

type OpportunityProjection struct {
	OpportunityID string         `json:"opportunity_id"`
	FeedbackCount int            `json:"feedback_count"`
	LatestMetrics map[string]any `json:"latest_metrics"`
	UpdatedAt     *time.Time     `json:"updated_at,omitempty"`
}

type Overview struct {
	Feedback    []OutcomeFeedback       `json:"feedback"`
	Projections []OpportunityProjection `json:"projections"`
}
