package application

import (
	"context"
	"fmt"
	"sync"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type MemoryStore struct {
	opportunities *opportunity.Service
	audit         *audit.Log
	mu            sync.RWMutex
	incubations   map[string]incubation.Project
	blueprints    map[string]blueprint.BusinessBlueprint
}

func NewMemoryStore() *MemoryStore {
	auditLog := &audit.Log{}
	return &MemoryStore{
		opportunities: opportunity.NewService(opportunity.NewMemoryRepository(), auditLog, &outbox.Memory{}),
		audit:         auditLog,
		incubations:   map[string]incubation.Project{},
		blueprints:    map[string]blueprint.BusinessBlueprint{},
	}
}

func (s *MemoryStore) CreateOpportunity(_ context.Context, scope tenancy.Scope, name, description, key string) (opportunity.Opportunity, error) {
	return s.opportunities.Create(scope, name, description, key)
}
func (s *MemoryStore) ListOpportunities(_ context.Context, scope tenancy.Scope) ([]opportunity.Opportunity, error) {
	return s.opportunities.List(scope), nil
}
func (s *MemoryStore) GetOpportunity(_ context.Context, scope tenancy.Scope, id string) (opportunity.Opportunity, error) {
	return s.opportunities.Get(scope, id)
}
func (s *MemoryStore) AddEvidence(_ context.Context, scope tenancy.Scope, id string, evidence opportunity.Evidence, key string) (opportunity.Opportunity, error) {
	return s.opportunities.AddEvidence(scope, id, evidence, key)
}
func (s *MemoryStore) ScoreOpportunity(_ context.Context, scope tenancy.Scope, id string, score int, key string) (opportunity.Opportunity, error) {
	return s.opportunities.Score(scope, id, score, key)
}
func (s *MemoryStore) TransitionOpportunity(_ context.Context, scope tenancy.Scope, id, to, key string) (opportunity.Opportunity, error) {
	return s.opportunities.Transition(scope, id, to, key)
}
func (s *MemoryStore) ReviewOpportunity(_ context.Context, scope tenancy.Scope, id, decision, rationale, key string) (opportunity.Opportunity, error) {
	item, err := s.opportunities.Get(scope, id)
	if err != nil {
		return opportunity.Opportunity{}, err
	}
	if item.Status != "under_review" {
		return opportunity.Opportunity{}, platform.Invalid("review_not_ready", "opportunity must be under review")
	}
	if decision != "approved" && decision != "rejected" {
		return opportunity.Opportunity{}, platform.Invalid("invalid_review_decision", "decision must be approved or rejected")
	}
	item, err = s.opportunities.Transition(scope, id, decision, key)
	if err == nil {
		s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "opportunity.review", ObjectType: "opportunity", ObjectID: id, RequestID: key, TraceID: scope.TraceID, Metadata: map[string]any{"decision": decision, "rationale": rationale}})
	}
	return item, err
}
func (s *MemoryStore) ListAudit(_ context.Context, scope tenancy.Scope) ([]audit.Record, error) {
	return s.audit.ForTenant(scope.TenantID), nil
}

func memoryKey(tenantID, id string) string { return tenantID + "/" + id }

func (s *MemoryStore) CreateIncubation(_ context.Context, scope tenancy.Scope, opportunityID, name, key string) (incubation.Project, error) {
	item, err := s.opportunities.Get(scope, opportunityID)
	if err != nil {
		return incubation.Project{}, err
	}
	if item.Status != "approved" {
		return incubation.Project{}, platform.Invalid("incubation_not_ready", "opportunity must be approved")
	}
	if name == "" {
		return incubation.Project{}, platform.Invalid("invalid_name", "incubation name is required")
	}
	if _, err = s.opportunities.Transition(scope, opportunityID, "incubating", key+":opportunity"); err != nil {
		return incubation.Project{}, err
	}
	project := incubation.New(scope.TenantID, opportunityID, name)
	s.mu.Lock()
	s.incubations[memoryKey(scope.TenantID, project.ID)] = project
	s.mu.Unlock()
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "incubation.create", ObjectType: "incubation_project", ObjectID: project.ID, RequestID: key, TraceID: scope.TraceID})
	return project, nil
}
func (s *MemoryStore) ListIncubations(_ context.Context, scope tenancy.Scope) ([]incubation.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []incubation.Project{}
	for _, item := range s.incubations {
		if item.TenantID == scope.TenantID {
			items = append(items, item)
		}
	}
	return items, nil
}
func (s *MemoryStore) TransitionIncubation(_ context.Context, scope tenancy.Scope, id, to, key string) (incubation.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := memoryKey(scope.TenantID, id)
	item, ok := s.incubations[mapKey]
	if !ok {
		return incubation.Project{}, fmt.Errorf("incubation project not found")
	}
	if err := item.Transition(to); err != nil {
		return incubation.Project{}, err
	}
	s.incubations[mapKey] = item
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "incubation.transition", ObjectType: "incubation_project", ObjectID: id, RequestID: key, TraceID: scope.TraceID, Metadata: map[string]any{"to": to}})
	return item, nil
}

func (s *MemoryStore) CreateBlueprint(_ context.Context, scope tenancy.Scope, opportunityID string, input BlueprintInput, key string) (blueprint.BusinessBlueprint, error) {
	if _, err := s.opportunities.Get(scope, opportunityID); err != nil {
		return blueprint.BusinessBlueprint{}, err
	}
	item := BuildBlueprint(scope, opportunityID, input)
	if item.Name == "" {
		return blueprint.BusinessBlueprint{}, platform.Invalid("invalid_name", "blueprint name is required")
	}
	s.mu.Lock()
	s.blueprints[memoryKey(scope.TenantID, item.ID)] = item
	s.mu.Unlock()
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "blueprint.create", ObjectType: "business_blueprint", ObjectID: item.ID, RequestID: key, TraceID: scope.TraceID})
	return item, nil
}
func (s *MemoryStore) ListBlueprints(_ context.Context, scope tenancy.Scope) ([]blueprint.BusinessBlueprint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []blueprint.BusinessBlueprint{}
	for _, item := range s.blueprints {
		if item.TenantID == scope.TenantID {
			items = append(items, item)
		}
	}
	return items, nil
}
func (s *MemoryStore) TransitionBlueprint(_ context.Context, scope tenancy.Scope, id, to, key string) (blueprint.BusinessBlueprint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := memoryKey(scope.TenantID, id)
	item, ok := s.blueprints[mapKey]
	if !ok {
		return blueprint.BusinessBlueprint{}, fmt.Errorf("blueprint not found")
	}
	if to == "approved" {
		if err := item.ValidateCompleteness(); err != nil {
			return blueprint.BusinessBlueprint{}, platform.Invalid("blueprint_incomplete", err.Error())
		}
	}
	if err := item.Transition(to, scope.ActorID); err != nil {
		return blueprint.BusinessBlueprint{}, err
	}
	s.blueprints[mapKey] = item
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "blueprint.transition", ObjectType: "business_blueprint", ObjectID: id, RequestID: key, TraceID: scope.TraceID, Metadata: map[string]any{"to": to}})
	return item, nil
}

func BuildBlueprint(scope tenancy.Scope, opportunityID string, input BlueprintInput) blueprint.BusinessBlueprint {
	item := blueprint.New(scope.TenantID, scope.ActorID, opportunityID, input.Name, input.Description)
	item.ValueProposition = input.ValueProposition
	item.RequiredCapabilities = input.RequiredCapabilities
	item.ProductDefinitions = input.ProductDefinitions
	item.WorkflowDefinitions = input.WorkflowDefinitions
	item.MeteringDefinitions = input.MeteringDefinitions
	item.PricingDefinitions = input.PricingDefinitions
	item.ComplianceProfile = input.ComplianceProfile
	item.BusinessModel = input.BusinessModel
	item.TargetMarketDefinition = input.TargetMarketDefinition
	item.RevenueModel = input.RevenueModel
	item.DeliveryModel = input.DeliveryModel
	return item
}
