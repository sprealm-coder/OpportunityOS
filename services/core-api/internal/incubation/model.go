package incubation

import (
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
	"time"
)

type Project struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	OpportunityID string    `json:"opportunity_id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type Experiment struct {
	ID, TenantID, ProjectID, Name, Hypothesis, Status string
	SuccessMetric                                     map[string]any
}
type Decision struct {
	ID, TenantID, ProjectID, ActorID, Decision, Rationale string
	CreatedAt                                             time.Time
}

func New(tenantID, opportunityID, name string) Project {
	now := time.Now().UTC()
	return Project{ID: platform.NewID("inc"), TenantID: tenantID, OpportunityID: opportunityID, Name: name, Status: "draft", Version: 1, CreatedAt: now, UpdatedAt: now}
}
func (p *Project) Transition(to string) error {
	if err := state.Incubation.Transition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	p.Version++
	p.UpdatedAt = time.Now().UTC()
	return nil
}
