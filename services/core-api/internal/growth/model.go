package growth

import (
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type MarketSegment struct {
	ID, TenantID, Name string
	Definition         map[string]any
}
type Lead struct {
	ID, TenantID, SegmentID, Name, Status string
	Evidence                              []string
	CreatedAt                             time.Time
}
type ProofTemplate struct {
	ID, TenantID, Name, Type, WorkflowVersionID string
	InputSchema, OutputSchema                   map[string]any
}
type ProofRequest struct {
	ID, TenantID, LeadID, DealID, TemplateID, Status, ArtifactID string
	ExpiresAt                                                    time.Time
}
type Deal struct {
	ID, TenantID, LeadID, Status string
	ValueMinor                   int64
	Currency                     string
}
type Quote struct {
	ID, TenantID, DealID, PriceVersionID, Status string
	AmountMinor                                  int64
	Currency                                     string
}

var proofTypes = map[string]bool{"report": true, "sample": true, "comparison": true, "prototype": true, "analysis": true, "audit": true, "simulation": true, "document": true, "media": true, "custom": true}

func NewLead(tenantID, segmentID, name string) Lead {
	return Lead{ID: platform.NewID("lead"), TenantID: tenantID, SegmentID: segmentID, Name: name, Status: "discovered", CreatedAt: time.Now().UTC()}
}
func (l *Lead) Transition(to string) error {
	if err := state.Lead.Transition(l.Status, to); err != nil {
		return err
	}
	l.Status = to
	return nil
}
func (p ProofTemplate) Valid() bool { return proofTypes[p.Type] && p.WorkflowVersionID != "" }
