package opportunity

import "time"

type Source struct {
	ID, TenantID, Name, ConnectorType string
	CreatedAt                         time.Time
}

type Signal struct {
	ID, TenantID, SourceID, Fingerprint string
	Payload                             map[string]any
	CreatedAt                           time.Time
}

type Evidence struct {
	ID, TenantID, OpportunityID, Kind, Summary string
	Confidence                                 int
	CreatedAt                                  time.Time
}

type Opportunity struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Score       int        `json:"score"`
	Version     int        `json:"version"`
	Evidence    []Evidence `json:"evidence"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
