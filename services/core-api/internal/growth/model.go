package growth

import (
	"time"

	orderdomain "github.com/opportunity-os/opportunity-os/services/core-api/internal/order"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type MarketSegment struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Definition map[string]any `json:"definition"`
	Version    int            `json:"version"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type ICPDefinition struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	MarketSegmentID string         `json:"market_segment_id"`
	Name            string         `json:"name"`
	Status          string         `json:"status"`
	Definition      map[string]any `json:"definition"`
	Version         int            `json:"version"`
	CreatedBy       string         `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Lead struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	SegmentID       string    `json:"market_segment_id"`
	ICPDefinitionID string    `json:"icp_definition_id,omitempty"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	Score           int       `json:"score"`
	Version         int       `json:"version"`
	Evidence        []string  `json:"evidence,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LeadEvidence struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	LeadID     string    `json:"lead_id"`
	Kind       string    `json:"kind"`
	Summary    string    `json:"summary"`
	Confidence int       `json:"confidence"`
	SourceRef  string    `json:"source_ref"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type Contact struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	LeadID          string    `json:"lead_id"`
	Channel         string    `json:"channel"`
	Value           string    `json:"value"`
	NormalizedValue string    `json:"normalized_value"`
	Status          string    `json:"status"`
	ConsentStatus   string    `json:"consent_status"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ContactSource struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	ContactID  string         `json:"contact_id"`
	SourceType string         `json:"source_type"`
	SourceRef  string         `json:"source_ref"`
	Evidence   map[string]any `json:"evidence"`
	CreatedAt  time.Time      `json:"created_at"`
}

type ProofTemplate struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	Name              string         `json:"name"`
	Type              string         `json:"proof_type"`
	WorkflowVersionID string         `json:"workflow_version_id"`
	InputSchema       map[string]any `json:"input_schema"`
	OutputSchema      map[string]any `json:"output_schema"`
	AccessPolicy      map[string]any `json:"access_policy"`
	RetentionDays     int            `json:"retention_days"`
	Status            string         `json:"status"`
	Version           int            `json:"version"`
	CreatedBy         string         `json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type ProofRequest struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	LeadID      string         `json:"lead_id"`
	DealID      string         `json:"deal_id,omitempty"`
	TemplateID  string         `json:"template_id"`
	Status      string         `json:"status"`
	ArtifactID  string         `json:"artifact_id,omitempty"`
	Input       map[string]any `json:"input"`
	RequestedBy string         `json:"requested_by"`
	ExpiresAt   time.Time      `json:"expires_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ProofInstance struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	ProofRequestID  string         `json:"proof_request_id"`
	Status          string         `json:"status"`
	Result          map[string]any `json:"result"`
	ArtifactRef     string         `json:"artifact_ref"`
	ReviewRationale string         `json:"review_rationale"`
	GeneratedBy     string         `json:"generated_by"`
	ReviewedBy      string         `json:"reviewed_by,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	ReviewedAt      *time.Time     `json:"reviewed_at,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at"`
}

type Campaign struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	MarketSegmentID string            `json:"market_segment_id,omitempty"`
	Name            string            `json:"name"`
	Channel         string            `json:"channel"`
	Purpose         string            `json:"purpose"`
	Status          string            `json:"status"`
	Version         int               `json:"version"`
	CreatedBy       string            `json:"created_by"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Steps           []CampaignStep    `json:"steps"`
	Approval        *CampaignApproval `json:"approval,omitempty"`
}

type CampaignStep struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	CampaignID string         `json:"campaign_id"`
	Position   int            `json:"position"`
	Kind       string         `json:"kind"`
	Definition map[string]any `json:"definition"`
	CreatedAt  time.Time      `json:"created_at"`
}

type CampaignApproval struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	CampaignID      string    `json:"campaign_id"`
	CampaignVersion int       `json:"campaign_version"`
	Decision        string    `json:"decision"`
	Rationale       string    `json:"rationale"`
	ReviewedBy      string    `json:"reviewed_by"`
	CreatedAt       time.Time `json:"created_at"`
}

type SuppressionEntry struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	Channel     string     `json:"channel"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	SourceRef   string     `json:"source_ref"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ReleasedAt  *time.Time `json:"released_at,omitempty"`
}

type SendQuota struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Channel       string    `json:"channel"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	LimitCount    int       `json:"limit_count"`
	ReservedCount int       `json:"reserved_count"`
	ConsumedCount int       `json:"consumed_count"`
}

type OutreachMessage struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	CampaignID        string         `json:"campaign_id"`
	CampaignStepID    string         `json:"campaign_step_id"`
	LeadID            string         `json:"lead_id"`
	ContactID         string         `json:"contact_id,omitempty"`
	Status            string         `json:"status"`
	Content           map[string]any `json:"content"`
	BlockReason       string         `json:"block_reason,omitempty"`
	ExternalMessageID string         `json:"external_message_id,omitempty"`
	IdempotencyKey    string         `json:"idempotency_key"`
	CreatedBy         string         `json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type Conversation struct {
	ID            string                `json:"id"`
	TenantID      string                `json:"tenant_id"`
	LeadID        string                `json:"lead_id"`
	DealID        string                `json:"deal_id,omitempty"`
	Channel       string                `json:"channel"`
	Status        string                `json:"status"`
	CreatedBy     string                `json:"created_by"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	LastMessageAt *time.Time            `json:"last_message_at,omitempty"`
	Messages      []ConversationMessage `json:"messages"`
}

type ConversationMessage struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	ConversationID string         `json:"conversation_id"`
	Direction      string         `json:"direction"`
	Status         string         `json:"status"`
	Content        map[string]any `json:"content"`
	IdempotencyKey string         `json:"idempotency_key"`
	CreatedBy      string         `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
}

type Deal struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	LeadID     string     `json:"lead_id"`
	Name       string     `json:"name"`
	CustomerID string     `json:"customer_id"`
	Status     string     `json:"status"`
	ValueMinor int64      `json:"value_minor"`
	Currency   string     `json:"currency"`
	Version    int        `json:"version"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
}

type Experiment struct {
	ID                    string         `json:"id"`
	TenantID              string         `json:"tenant_id"`
	Name                  string         `json:"name"`
	EntityType            string         `json:"entity_type"`
	EntityID              string         `json:"entity_id"`
	Hypothesis            string         `json:"hypothesis"`
	Status                string         `json:"status"`
	AllocationBasisPoints int            `json:"allocation_basis_points"`
	MetricsDefinition     map[string]any `json:"metrics_definition"`
	Result                map[string]any `json:"result"`
	Version               int            `json:"version"`
	CreatedBy             string         `json:"created_by"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type Overview struct {
	Segments       []MarketSegment    `json:"segments"`
	ICPs           []ICPDefinition    `json:"icps"`
	Leads          []Lead             `json:"leads"`
	Evidence       []LeadEvidence     `json:"evidence"`
	Contacts       []Contact          `json:"contacts"`
	ProofTemplates []ProofTemplate    `json:"proof_templates"`
	ProofRequests  []ProofRequest     `json:"proof_requests"`
	ProofInstances []ProofInstance    `json:"proof_instances"`
	Campaigns      []Campaign         `json:"campaigns"`
	Suppressions   []SuppressionEntry `json:"suppressions"`
	Outreach       []OutreachMessage  `json:"outreach"`
	Conversations  []Conversation     `json:"conversations"`
	Deals          []Deal             `json:"deals"`
	Experiments    []Experiment       `json:"experiments"`
}

type Quote = orderdomain.Quote

var proofTypes = map[string]bool{"report": true, "sample": true, "comparison": true, "prototype": true, "analysis": true, "audit": true, "simulation": true, "document": true, "media": true, "custom": true}

func NewLead(tenantID, segmentID, name string) Lead {
	now := time.Now().UTC()
	return Lead{ID: platform.NewID("lead"), TenantID: tenantID, SegmentID: segmentID, Name: name, Status: "discovered", Version: 1, CreatedAt: now, UpdatedAt: now}
}

func (l *Lead) Transition(to string) error {
	if err := state.Lead.Transition(l.Status, to); err != nil {
		return err
	}
	l.Status = to
	l.Version++
	l.UpdatedAt = time.Now().UTC()
	return nil
}

func (p ProofTemplate) Valid() bool { return proofTypes[p.Type] && p.WorkflowVersionID != "" }

func (p *ProofRequest) Transition(to string) error {
	if err := state.ProofRequest.Transition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Campaign) Transition(to string) error {
	if err := state.Campaign.Transition(c.Status, to); err != nil {
		return err
	}
	c.Status = to
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (d *Deal) Transition(to string) error {
	if err := state.Deal.Transition(d.Status, to); err != nil {
		return err
	}
	d.Status = to
	d.Version++
	d.UpdatedAt = time.Now().UTC()
	if to == "won" || to == "lost" || to == "cancelled" {
		now := time.Now().UTC()
		d.ClosedAt = &now
	}
	return nil
}

func (e *Experiment) Transition(to string) error {
	if err := state.Experiment.Transition(e.Status, to); err != nil {
		return err
	}
	e.Status = to
	e.Version++
	e.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *OutreachMessage) Transition(to string) error {
	if err := state.Outreach.Transition(m.Status, to); err != nil {
		return err
	}
	m.Status = to
	m.UpdatedAt = time.Now().UTC()
	return nil
}
