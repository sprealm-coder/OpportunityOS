package intelligence

import "time"

type Source struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	Name          string         `json:"name"`
	ConnectorType string         `json:"connector_type"`
	Status        string         `json:"status"`
	Config        map[string]any `json:"config"`
	Version       int            `json:"version"`
	CreatedBy     string         `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Signal struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	SourceID      string         `json:"source_id"`
	OpportunityID string         `json:"opportunity_id,omitempty"`
	ExternalID    string         `json:"external_id,omitempty"`
	Fingerprint   string         `json:"fingerprint"`
	Status        string         `json:"status"`
	Payload       map[string]any `json:"payload"`
	Normalized    map[string]any `json:"normalized"`
	OccurredAt    time.Time      `json:"occurred_at"`
	ImportedBy    string         `json:"imported_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Overview struct {
	Sources []Source `json:"sources"`
	Signals []Signal `json:"signals"`
}
