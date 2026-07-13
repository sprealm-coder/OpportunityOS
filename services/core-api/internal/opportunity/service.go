package opportunity

import (
	"fmt"
	"sync"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type Repository interface {
	Create(Opportunity) error
	Get(tenantID, id string) (Opportunity, error)
	Save(Opportunity) error
	List(tenantID string) []Opportunity
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]Opportunity
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]Opportunity)}
}
func repoKey(tenantID, id string) string { return tenantID + "/" + id }
func (r *MemoryRepository) Create(item Opportunity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := repoKey(item.TenantID, item.ID)
	if _, ok := r.items[key]; ok {
		return fmt.Errorf("opportunity exists")
	}
	r.items[key] = item
	return nil
}
func (r *MemoryRepository) Get(tenantID, id string) (Opportunity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[repoKey(tenantID, id)]
	if !ok {
		return Opportunity{}, fmt.Errorf("opportunity not found")
	}
	return item, nil
}
func (r *MemoryRepository) Save(item Opportunity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := repoKey(item.TenantID, item.ID)
	if _, ok := r.items[key]; !ok {
		return fmt.Errorf("opportunity not found")
	}
	r.items[key] = item
	return nil
}
func (r *MemoryRepository) List(tenantID string) []Opportunity {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []Opportunity{}
	for _, item := range r.items {
		if item.TenantID == tenantID {
			out = append(out, item)
		}
	}
	return out
}

type Service struct {
	repo   Repository
	audit  *audit.Log
	outbox *outbox.Memory
}

func NewService(repo Repository, auditLog *audit.Log, events *outbox.Memory) *Service {
	return &Service{repo: repo, audit: auditLog, outbox: events}
}

func (s *Service) Create(scope tenancy.Scope, name, description, requestID string) (Opportunity, error) {
	if scope.TenantID == "" {
		return Opportunity{}, platform.ErrTenantRequired
	}
	if name == "" {
		return Opportunity{}, platform.Invalid("invalid_name", "opportunity name is required")
	}
	now := time.Now().UTC()
	item := Opportunity{ID: platform.NewID("opp"), TenantID: scope.TenantID, Name: name, Description: description, Status: "detected", Version: 1, CreatedBy: scope.ActorID, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(item); err != nil {
		return Opportunity{}, err
	}
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "opportunity.create", ObjectType: "opportunity", ObjectID: item.ID, RequestID: requestID})
	s.outbox.Append(outbox.Event{TenantID: scope.TenantID, AggregateType: "opportunity", AggregateID: item.ID, EventType: "opportunity.created", Version: item.Version, TraceID: scope.TraceID, Payload: map[string]any{"status": item.Status}})
	return item, nil
}

func (s *Service) AddEvidence(scope tenancy.Scope, id string, evidence Evidence, requestID string) (Opportunity, error) {
	item, err := s.repo.Get(scope.TenantID, id)
	if err != nil {
		return Opportunity{}, err
	}
	evidence.ID = platform.NewID("evd")
	evidence.TenantID = scope.TenantID
	evidence.OpportunityID = id
	evidence.CreatedAt = time.Now().UTC()
	item.Evidence = append(item.Evidence, evidence)
	item.Version++
	item.UpdatedAt = time.Now().UTC()
	if item.Status == "detected" {
		if err := state.Opportunity.Transition(item.Status, "enriched"); err != nil {
			return Opportunity{}, err
		}
		item.Status = "enriched"
	}
	if err := s.repo.Save(item); err != nil {
		return Opportunity{}, err
	}
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "opportunity.add_evidence", ObjectType: "opportunity", ObjectID: id, RequestID: requestID})
	return item, nil
}

func (s *Service) Score(scope tenancy.Scope, id string, score int, requestID string) (Opportunity, error) {
	if score < 0 || score > 100 {
		return Opportunity{}, platform.Invalid("invalid_score", "score must be between 0 and 100")
	}
	item, err := s.repo.Get(scope.TenantID, id)
	if err != nil {
		return Opportunity{}, err
	}
	if item.Status != "enriched" {
		return Opportunity{}, fmt.Errorf("opportunity must be enriched before scoring")
	}
	if err := state.Opportunity.Transition(item.Status, "scored"); err != nil {
		return Opportunity{}, err
	}
	item.Score = score
	item.Status = "scored"
	item.Version++
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.Save(item); err != nil {
		return Opportunity{}, err
	}
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "opportunity.score", ObjectType: "opportunity", ObjectID: id, RequestID: requestID, Metadata: map[string]any{"score": score}})
	return item, nil
}

func (s *Service) Transition(scope tenancy.Scope, id, to, requestID string) (Opportunity, error) {
	item, err := s.repo.Get(scope.TenantID, id)
	if err != nil {
		return Opportunity{}, err
	}
	if err := state.Opportunity.Transition(item.Status, to); err != nil {
		return Opportunity{}, err
	}
	from := item.Status
	item.Status = to
	item.Version++
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.Save(item); err != nil {
		return Opportunity{}, err
	}
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "opportunity.transition", ObjectType: "opportunity", ObjectID: id, RequestID: requestID, Metadata: map[string]any{"from": from, "to": to}})
	s.outbox.Append(outbox.Event{TenantID: scope.TenantID, AggregateType: "opportunity", AggregateID: id, EventType: "opportunity.transitioned", Version: item.Version, TraceID: scope.TraceID, Payload: map[string]any{"from": from, "to": to}})
	return item, nil
}

func (s *Service) Get(scope tenancy.Scope, id string) (Opportunity, error) {
	return s.repo.Get(scope.TenantID, id)
}
func (s *Service) List(scope tenancy.Scope) []Opportunity { return s.repo.List(scope.TenantID) }
