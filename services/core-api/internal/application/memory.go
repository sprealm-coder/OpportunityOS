package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/capability"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type MemoryStore struct {
	opportunities *opportunity.Service
	audit         *audit.Log
	mu            sync.RWMutex
	incubations   map[string]incubation.Project
	blueprints    map[string]blueprint.BusinessBlueprint
	capabilities  map[string]capability.Capability
	providers     map[string]capability.Provider
	products      map[string]catalog.ProductDetail
	sessions      map[string]auth.Session
	passwordHash  string
}

func NewMemoryStore() *MemoryStore {
	auditLog := &audit.Log{}
	passwordHash, err := auth.HashPassword("opportunity-dev")
	if err != nil {
		panic(err)
	}
	return &MemoryStore{
		opportunities: opportunity.NewService(opportunity.NewMemoryRepository(), auditLog, &outbox.Memory{}),
		audit:         auditLog,
		incubations:   map[string]incubation.Project{},
		blueprints:    map[string]blueprint.BusinessBlueprint{},
		capabilities:  map[string]capability.Capability{},
		providers:     map[string]capability.Provider{},
		products:      map[string]catalog.ProductDetail{},
		sessions:      map[string]auth.Session{},
		passwordHash:  passwordHash,
	}
}

func (s *MemoryStore) CreateSession(_ context.Context, email, password string) (auth.Session, error) {
	if email != "admin@opportunity.local" || auth.VerifyPassword(s.passwordHash, password) != nil {
		return auth.Session{}, platform.Invalid("invalid_credentials", "email or password is incorrect")
	}
	token, err := auth.NewToken()
	if err != nil {
		return auth.Session{}, err
	}
	session := auth.Session{
		ID: "session-memory", Token: token, UserID: "user-memory", Email: email,
		DisplayName: "Test Admin", TenantID: "tenant-api-flow", TenantName: "Test Tenant",
		Role: "admin", ExpiresAt: time.Now().UTC().Add(12 * time.Hour),
	}
	s.mu.Lock()
	s.sessions[auth.HashToken(token)] = session
	s.mu.Unlock()
	return session, nil
}

func (s *MemoryStore) ResolveSession(_ context.Context, token string) (auth.Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[auth.HashToken(token)]
	s.mu.RUnlock()
	if !ok || time.Now().UTC().After(session.ExpiresAt) {
		return auth.Session{}, platform.Invalid("session_invalid", "session is missing or expired")
	}
	return session, nil
}

func (s *MemoryStore) RevokeSession(_ context.Context, token string) error {
	s.mu.Lock()
	delete(s.sessions, auth.HashToken(token))
	s.mu.Unlock()
	return nil
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

func (s *MemoryStore) CreateCapability(_ context.Context, scope tenancy.Scope, name, description string, definition map[string]any, key string) (capability.Capability, error) {
	if name == "" {
		return capability.Capability{}, platform.Invalid("invalid_name", "capability name is required")
	}
	item := capability.New(scope.TenantID, name)
	item.Description, item.Definition = description, definition
	s.mu.Lock()
	s.capabilities[memoryKey(scope.TenantID, item.ID)] = item
	s.mu.Unlock()
	s.audit.Append(audit.Record{TenantID: scope.TenantID, ActorID: scope.ActorID, Action: "capability.create", ObjectType: "capability", ObjectID: item.ID, RequestID: key})
	return item, nil
}

func (s *MemoryStore) ListCapabilities(_ context.Context, scope tenancy.Scope) ([]capability.Capability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []capability.Capability{}
	for _, item := range s.capabilities {
		if item.TenantID == scope.TenantID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) CreateProvider(_ context.Context, scope tenancy.Scope, name, key string) (capability.Provider, error) {
	if name == "" {
		return capability.Provider{}, platform.Invalid("invalid_name", "provider name is required")
	}
	item := capability.NewProvider(scope.TenantID, name)
	s.mu.Lock()
	s.providers[memoryKey(scope.TenantID, item.ID)] = item
	s.mu.Unlock()
	return item, nil
}

func (s *MemoryStore) ListProviders(_ context.Context, scope tenancy.Scope) ([]capability.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []capability.Provider{}
	for _, item := range s.providers {
		if item.TenantID == scope.TenantID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) CreateProviderEndpoint(_ context.Context, scope tenancy.Scope, providerID, capabilityID, adapterType, adapterVersion, key string) (capability.ProviderEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	providerKey := memoryKey(scope.TenantID, providerID)
	provider, ok := s.providers[providerKey]
	if !ok {
		return capability.ProviderEndpoint{}, platform.Invalid("not_found", "provider not found")
	}
	if _, ok = s.capabilities[memoryKey(scope.TenantID, capabilityID)]; !ok {
		return capability.ProviderEndpoint{}, platform.Invalid("not_found", "capability not found")
	}
	item := capability.NewEndpoint(scope.TenantID, providerID, capabilityID, adapterType)
	if adapterVersion != "" {
		item.AdapterVersion = adapterVersion
	}
	provider.Endpoints = append(provider.Endpoints, item)
	s.providers[providerKey] = provider
	return item, nil
}

func (s *MemoryStore) CreateProduct(_ context.Context, scope tenancy.Scope, blueprintID, name, key string) (catalog.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bp, ok := s.blueprints[memoryKey(scope.TenantID, blueprintID)]
	if !ok || (bp.Status != "approved" && bp.Status != "configuring" && bp.Status != "ready" && bp.Status != "launched") {
		return catalog.Product{}, platform.Invalid("product_blueprint_not_ready", "blueprint must be approved before product creation")
	}
	item, _ := catalog.Draft(scope.TenantID, blueprintID, name)
	item.CreatedBy = scope.ActorID
	s.products[memoryKey(scope.TenantID, item.ID)] = catalog.ProductDetail{Product: item, Versions: []catalog.ProductVersion{}, SKUs: []catalog.SKUDetail{}, Publications: []catalog.Publication{}}
	return item, nil
}

func (s *MemoryStore) ListProducts(_ context.Context, scope tenancy.Scope) ([]catalog.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []catalog.Product{}
	for _, detail := range s.products {
		if detail.TenantID == scope.TenantID {
			items = append(items, detail.Product)
		}
	}
	return items, nil
}

func (s *MemoryStore) GetProduct(_ context.Context, scope tenancy.Scope, id string) (catalog.ProductDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.products[memoryKey(scope.TenantID, id)]
	if !ok {
		return catalog.ProductDetail{}, platform.Invalid("not_found", "product not found")
	}
	return item, nil
}

func (s *MemoryStore) CreateProductVersion(_ context.Context, scope tenancy.Scope, id string, input ProductVersionInput, key string) (catalog.ProductVersion, error) {
	input, err := NormalizeProductVersionInput(scope, input)
	if err != nil {
		return catalog.ProductVersion{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := memoryKey(scope.TenantID, id)
	detail, ok := s.products[mapKey]
	if !ok {
		return catalog.ProductVersion{}, platform.Invalid("not_found", "product not found")
	}
	input.Workflow.ID = platform.NewID("workflow")
	input.Metering.ID = platform.NewID("meter")
	input.PriceBook.ID = platform.NewID("price")
	input.RoutePolicy.ID = platform.NewID("route")
	version := catalog.ProductVersion{ID: platform.NewID("prodv"), TenantID: scope.TenantID, ProductID: id, Version: len(detail.Versions) + 1, InputSchema: input.InputSchema, OutputSchema: input.OutputSchema, FormSchema: input.FormSchema, CapabilityIDs: input.CapabilityIDs, Workflow: input.Workflow, MeteringID: input.Metering.ID, PriceBookID: input.PriceBook.ID, RoutePolicyID: input.RoutePolicy.ID, DeliveryMode: input.DeliveryMode, ComplianceProfileID: platform.NewID("compliance"), ComplianceProfile: input.ComplianceProfile, GrowthPlaybook: input.GrowthPlaybook, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC()}
	detail.Versions = append(detail.Versions, version)
	detail.Version++
	detail.UpdatedAt = time.Now().UTC()
	s.products[mapKey] = detail
	return version, nil
}

func (s *MemoryStore) CreateSKU(_ context.Context, scope tenancy.Scope, productID, code, name, key string) (catalog.SKU, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := memoryKey(scope.TenantID, productID)
	detail, ok := s.products[mapKey]
	if !ok {
		return catalog.SKU{}, platform.Invalid("not_found", "product not found")
	}
	item := catalog.SKU{ID: platform.NewID("sku"), TenantID: scope.TenantID, ProductID: productID, Code: code, Name: name, Status: "draft", CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC()}
	detail.SKUs = append(detail.SKUs, catalog.SKUDetail{SKU: item, Versions: []catalog.SKUVersion{}})
	s.products[mapKey] = detail
	return item, nil
}

func (s *MemoryStore) CreateSKUVersion(_ context.Context, scope tenancy.Scope, skuID string, input SKUVersionInput, key string) (catalog.SKUVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for mapKey, detail := range s.products {
		for skuIndex, sku := range detail.SKUs {
			if sku.ID != skuID || sku.TenantID != scope.TenantID {
				continue
			}
			var version catalog.ProductVersion
			for _, candidate := range detail.Versions {
				if candidate.ID == input.ProductVersionID {
					version = candidate
				}
			}
			if version.ID == "" {
				return catalog.SKUVersion{}, platform.Invalid("invalid_reference", "product version does not belong to SKU product")
			}
			item := catalog.SKUVersion{ID: platform.NewID("skuv"), TenantID: scope.TenantID, SKUID: skuID, ProductVersionID: version.ID, WorkflowVersionID: version.Workflow.ID, MeteringVersionID: version.MeteringID, PricingVersionID: version.PriceBookID, RoutingVersionID: version.RoutePolicyID, Version: len(sku.Versions) + 1, Entitlements: input.Entitlements, CreatedBy: scope.ActorID, CreatedAt: time.Now().UTC()}
			detail.SKUs[skuIndex].Versions = append(detail.SKUs[skuIndex].Versions, item)
			s.products[mapKey] = detail
			return item, nil
		}
	}
	return catalog.SKUVersion{}, platform.Invalid("not_found", "SKU not found")
}

func (s *MemoryStore) PublishProduct(_ context.Context, scope tenancy.Scope, productID, productVersionID, key string) (catalog.Publication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mapKey := memoryKey(scope.TenantID, productID)
	detail, ok := s.products[mapKey]
	if !ok {
		return catalog.Publication{}, platform.Invalid("not_found", "product not found")
	}
	var version catalog.ProductVersion
	for _, candidate := range detail.Versions {
		if candidate.ID == productVersionID {
			version = candidate
		}
	}
	providers := map[string]bool{}
	for _, provider := range s.providers {
		if provider.TenantID == scope.TenantID && provider.Status == "active" {
			for _, endpoint := range provider.Endpoints {
				providers[endpoint.CapabilityID] = endpoint.Status == "healthy"
			}
		}
	}
	skuReady := false
	for _, sku := range detail.SKUs {
		for _, skuVersion := range sku.Versions {
			if skuVersion.ProductVersionID == productVersionID {
				skuReady = true
			}
		}
	}
	if !skuReady {
		return catalog.Publication{}, platform.Invalid("publication_not_ready", "at least one SKU version must bind the product version")
	}
	publication, err := catalog.Publish(&detail.Product, version, providers)
	if err != nil {
		return catalog.Publication{}, platform.Invalid("publication_not_ready", err.Error())
	}
	publication.PublishedBy = scope.ActorID
	detail.Publications = append(detail.Publications, publication)
	s.products[mapKey] = detail
	return publication, nil
}

func NormalizeProductVersionInput(scope tenancy.Scope, input ProductVersionInput) (ProductVersionInput, error) {
	if err := schema.Validate(input.InputSchema); err != nil {
		return input, platform.Invalid("invalid_input_schema", err.Error())
	}
	if err := schema.Validate(input.OutputSchema); err != nil {
		return input, platform.Invalid("invalid_output_schema", err.Error())
	}
	if len(input.FormSchema) > 0 {
		if err := schema.Validate(input.FormSchema); err != nil {
			return input, platform.Invalid("invalid_form_schema", err.Error())
		}
	} else {
		input.FormSchema = input.InputSchema
	}
	input.Workflow.TenantID = scope.TenantID
	if input.Workflow.Version == 0 {
		input.Workflow.Version = 1
	}
	if err := input.Workflow.Validate(); err != nil {
		return input, platform.Invalid("invalid_workflow", err.Error())
	}
	input.Metering.TenantID = scope.TenantID
	if input.Metering.Version == 0 {
		input.Metering.Version = 1
	}
	if err := input.Metering.Validate(); err != nil {
		return input, platform.Invalid("invalid_metering", err.Error())
	}
	input.PriceBook.TenantID = scope.TenantID
	if input.PriceBook.Version == 0 {
		input.PriceBook.Version = 1
	}
	if err := input.PriceBook.Validate(); err != nil {
		return input, platform.Invalid("invalid_pricing", err.Error())
	}
	input.RoutePolicy.TenantID = scope.TenantID
	if input.RoutePolicy.Version == 0 {
		input.RoutePolicy.Version = 1
	}
	if input.RoutePolicy.Strategy != "priority" && input.RoutePolicy.Strategy != "lowest_cost" {
		return input, platform.Invalid("invalid_routing", "route strategy must be priority or lowest_cost")
	}
	if len(input.CapabilityIDs) == 0 || input.RoutePolicy.Strategy == "" || input.DeliveryMode == "" || len(input.ComplianceProfile) == 0 {
		return input, platform.Invalid("product_version_incomplete", "capabilities, routing, delivery, and compliance are required")
	}
	allowedDelivery := map[string]bool{"workflow": true, "realtime": true, "async": true, "provisioning": true, "manual": true}
	if !allowedDelivery[input.DeliveryMode] {
		return input, platform.Invalid("invalid_delivery_mode", "unsupported delivery mode")
	}
	if input.GrowthPlaybook == nil {
		input.GrowthPlaybook = map[string]any{}
	}
	return input, nil
}
