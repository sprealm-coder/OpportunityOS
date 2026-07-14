package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/permission"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type Server struct {
	store           application.Store
	limiter         *rateLimiter
	allowHeaderAuth bool
}

func New() http.Handler {
	return newHandler(application.NewMemoryStore(), true)
}

func NewWithStore(store application.Store) http.Handler {
	return newHandler(store, false)
}

func newHandler(store application.Store, allowHeaderAuth bool) http.Handler {
	server := &Server{store: store, limiter: newRateLimiter(120, time.Minute), allowHeaderAuth: allowHeaderAuth}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(server.requestContext)
	r.Use(server.securityHeaders)
	r.Use(server.cors)
	r.Use(server.limiter.Middleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "core-api"})
	})
	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/sessions", server.createSession)
		r.Group(func(r chi.Router) {
			r.Use(server.authenticate)
			r.Get("/auth/session", server.getSession)
			r.Delete("/auth/session", server.deleteSession)
			r.With(server.requirePermission(permission.OpportunityRead)).Get("/opportunities", server.listOpportunities)
			r.With(server.requirePermission(permission.OpportunityCreate)).Post("/opportunities", server.createOpportunity)
			r.With(server.requirePermission(permission.OpportunityRead)).Get("/opportunities/{id}", server.getOpportunity)
			r.With(server.requirePermission(permission.OpportunityEvidence)).Post("/opportunities/{id}/evidence", server.addEvidence)
			r.With(server.requirePermission(permission.OpportunityScore)).Post("/opportunities/{id}/score", server.scoreOpportunity)
			r.With(server.requirePermission(permission.OpportunitySubmit)).Post("/opportunities/{id}/transitions", server.transitionOpportunity)
			r.With(server.requirePermission(permission.OpportunityReview)).Post("/opportunities/{id}/reviews", server.reviewOpportunity)
			r.With(server.requirePermission(permission.IncubationWrite)).Post("/opportunities/{id}/incubations", server.createIncubation)
			r.With(server.requirePermission(permission.BlueprintWrite)).Post("/opportunities/{id}/blueprints", server.createBlueprint)
			r.With(server.requirePermission(permission.IncubationRead)).Get("/incubations", server.listIncubations)
			r.With(server.requirePermission(permission.IncubationWrite)).Post("/incubations/{id}/transitions", server.transitionIncubation)
			r.With(server.requirePermission(permission.BlueprintRead)).Get("/blueprints", server.listBlueprints)
			r.With(server.requirePermission(permission.BlueprintWrite)).Post("/blueprints/{id}/transitions", server.transitionBlueprint)
			r.With(server.requirePermission(permission.CapabilityRead)).Get("/capabilities", server.listCapabilities)
			r.With(server.requirePermission(permission.CapabilityWrite)).Post("/capabilities", server.createCapability)
			r.With(server.requirePermission(permission.ProviderRead)).Get("/providers", server.listProviders)
			r.With(server.requirePermission(permission.ProviderWrite)).Post("/providers", server.createProvider)
			r.With(server.requirePermission(permission.ProviderWrite)).Post("/providers/{id}/endpoints", server.createProviderEndpoint)
			r.With(server.requirePermission(permission.ProductRead)).Get("/products", server.listProducts)
			r.With(server.requirePermission(permission.ProductRead)).Get("/products/{id}", server.getProduct)
			r.With(server.requirePermission(permission.ProductWrite)).Post("/blueprints/{id}/products", server.createProduct)
			r.With(server.requirePermission(permission.ProductWrite)).Post("/products/{id}/versions", server.createProductVersion)
			r.With(server.requirePermission(permission.ProductWrite)).Post("/products/{id}/skus", server.createSKU)
			r.With(server.requirePermission(permission.ProductWrite)).Post("/skus/{id}/versions", server.createSKUVersion)
			r.With(server.requirePermission(permission.ProductPublish)).Post("/products/{id}/publications", server.publishProduct)
			r.With(server.requirePermission(permission.TransactionRead)).Get("/quotes", server.listQuotes)
			r.With(server.requirePermission(permission.QuoteWrite)).Post("/quotes", server.createQuote)
			r.With(server.requirePermission(permission.TransactionRead)).Get("/quotes/{id}", server.getQuote)
			r.With(server.requirePermission(permission.QuoteWrite)).Post("/quotes/{id}/transitions", server.transitionQuote)
			r.With(server.requirePermission(permission.TransactionRead)).Get("/orders", server.listOrders)
			r.With(server.requirePermission(permission.OrderWrite)).Post("/orders", server.createOrder)
			r.With(server.requirePermission(permission.TransactionRead)).Get("/orders/{id}", server.getOrder)
			r.With(server.requirePermission(permission.OrderWrite)).Post("/orders/{id}/transitions", server.transitionOrder)
			r.With(server.requirePermission(permission.ExecutionWrite)).Post("/executions/{id}/transitions", server.transitionExecution)
			r.With(server.requirePermission(permission.ExecutionWrite)).Post("/deliveries/{id}/transitions", server.transitionDelivery)
			r.With(server.requirePermission(permission.BillingWrite)).Post("/executions/{id}/usage", server.recordUsage)
			r.With(server.requirePermission(permission.BillingWrite)).Post("/executions/{id}/provider-costs", server.recordProviderCost)
			r.With(server.requirePermission(permission.BillingWrite)).Post("/executions/{id}/customer-charges", server.createCustomerCharge)
			r.With(server.requirePermission(permission.FinanceRead)).Get("/finance", server.listFinance)
			r.With(server.requirePermission(permission.WalletWrite)).Post("/wallets", server.createWallet)
			r.With(server.requirePermission(permission.FinanceAdjust)).Post("/wallets/{id}/adjustments", server.postWalletAdjustment)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/orders/{id}/holds", server.placeOrderHold)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/holds/{id}/releases", server.releaseHold)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/customer-charges/{id}/postings", server.postCustomerCharge)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/customer-charges/{id}/refunds", server.refundCustomerCharge)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/customer-charges/{id}/commissions", server.createCommission)
			r.With(server.requirePermission(permission.LedgerPost)).Post("/provider-costs/{id}/payables", server.createProviderPayable)
			r.With(server.requirePermission(permission.SettlementWrite)).Post("/settlements", server.createSettlement)
			r.With(server.requirePermission(permission.ReconciliationWrite)).Post("/reconciliation-runs", server.runReconciliation)
			r.With(server.requirePermission(permission.GrowthRead)).Get("/growth", server.listGrowth)
			r.With(server.requirePermission(permission.GrowthWrite)).Post("/market-segments", server.createMarketSegment)
			r.With(server.requirePermission(permission.GrowthWrite)).Post("/market-segments/{id}/icps", server.createICPDefinition)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/leads", server.createLead)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/leads/{id}/evidence", server.addLeadEvidence)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/leads/{id}/transitions", server.transitionLead)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/leads/{id}/contacts", server.createContact)
			r.With(server.requirePermission(permission.ProofWrite)).Post("/proof-templates", server.createProofTemplate)
			r.With(server.requirePermission(permission.ProofWrite)).Post("/leads/{id}/proof-requests", server.createProofRequest)
			r.With(server.requirePermission(permission.ProofWrite)).Post("/proof-requests/{id}/generate", server.generateProof)
			r.With(server.requirePermission(permission.ProofReview)).Post("/proof-requests/{id}/reviews", server.reviewProof)
			r.With(server.requirePermission(permission.CampaignWrite)).Post("/campaigns", server.createCampaign)
			r.With(server.requirePermission(permission.CampaignWrite)).Post("/campaigns/{id}/steps", server.addCampaignStep)
			r.With(server.requirePermission(permission.CampaignWrite)).Post("/campaigns/{id}/transitions", server.transitionCampaign)
			r.With(server.requirePermission(permission.CampaignApprove)).Post("/campaigns/{id}/reviews", server.reviewCampaign)
			r.With(server.requirePermission(permission.OutreachWrite)).Post("/suppressions", server.createSuppression)
			r.With(server.requirePermission(permission.OutreachWrite)).Post("/campaigns/{id}/outreach", server.planOutreach)
			r.With(server.requirePermission(permission.OutreachWrite)).Post("/outreach/{id}/transitions", server.transitionOutreach)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/conversations", server.createConversation)
			r.With(server.requirePermission(permission.LeadWrite)).Post("/conversations/{id}/messages", server.addConversationMessage)
			r.With(server.requirePermission(permission.DealWrite)).Post("/deals", server.createDeal)
			r.With(server.requirePermission(permission.GrowthRead)).Get("/deals/{id}", server.getDeal)
			r.With(server.requirePermission(permission.DealWrite)).Post("/deals/{id}/transitions", server.transitionDeal)
			r.With(server.requirePermission(permission.DealWrite), server.requirePermission(permission.QuoteWrite)).Post("/deals/{id}/quotes", server.createDealQuote)
			r.With(server.requirePermission(permission.ExperimentWrite)).Post("/experiments", server.createExperiment)
			r.With(server.requirePermission(permission.ExperimentWrite)).Post("/experiments/{id}/transitions", server.transitionExperiment)
			r.With(server.requirePermission(permission.AuditRead)).Get("/audit", server.listAudit)
		})
	})
	return r
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowHeaderAuth {
			tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			actorID := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
			if tenantID != "" && actorID != "" {
				scope := tenancy.Scope{TenantID: tenantID, ActorID: actorID, Role: "admin", TraceID: r.Header.Get("X-Trace-ID")}
				next.ServeHTTP(w, r.WithContext(tenancy.WithScope(r.Context(), scope)))
				return
			}
		}
		cookie, err := r.Cookie(auth.SessionCookie)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, platform.Invalid("authentication_required", "a valid session is required"))
			return
		}
		session, err := s.store.ResolveSession(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, platform.Invalid("authentication_required", "a valid session is required"))
			return
		}
		tenantScope := tenancy.Scope{TenantID: session.TenantID, ActorID: session.UserID, Role: session.Role, TraceID: r.Header.Get("X-Trace-ID")}
		next.ServeHTTP(w, r.WithContext(tenancy.WithScope(r.Context(), tenantScope)))
	})
}

func (s *Server) requirePermission(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantScope, err := scope(r)
			if err != nil {
				writeError(w, r, http.StatusUnauthorized, err)
				return
			}
			if err = permission.RequireRole(tenantScope.Role, required); err != nil {
				writeError(w, r, http.StatusForbidden, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = platform.NewID("req")
		}
		traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID"))
		if traceID == "" {
			traceID = platform.NewID("trace")
		}
		r.Header.Set("X-Request-ID", requestID)
		r.Header.Set("X-Trace-ID", traceID)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Trace-ID", traceID)
		next.ServeHTTP(w, r)
	})
}
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{
		"http://127.0.0.1:3000": true,
		"http://127.0.0.1:3001": true,
		"http://localhost:3000": true,
		"http://localhost:3001": true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Request-ID, X-Trace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &input) {
		return
	}
	session, err := s.store.CreateSession(r.Context(), input.Email, input.Password)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, platform.Invalid("invalid_credentials", "email or password is incorrect"))
		return
	}
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookie, Value: session.Token, Path: "/", Expires: session.ExpiresAt,
		MaxAge: maxAge, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookie)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, platform.Invalid("authentication_required", "a valid session is required"))
		return
	}
	session, err := s.store.ResolveSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, platform.Invalid("authentication_required", "a valid session is required"))
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		if err = s.store.RevokeSession(r.Context(), cookie.Value); err != nil {
			writeError(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func scope(r *http.Request) (tenancy.Scope, error) { return tenancy.FromContext(r.Context()) }
func idempotency(r *http.Request) (string, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return "", platform.ErrIdempotencyKeyRequired
	}
	return key, nil
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, r, http.StatusBadRequest, platform.Invalid("invalid_json", err.Error()))
		return false
	}
	return true
}

func (s *Server) createOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateOpportunity(r.Context(), tenantScope, input.Name, input.Description, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) listOpportunities(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListOpportunities(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) getOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	item, err := s.store.GetOpportunity(r.Context(), tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, platform.Invalid("not_found", "opportunity not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) addEvidence(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Kind       string `json:"kind"`
		Summary    string `json:"summary"`
		Confidence int    `json:"confidence"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.AddEvidence(r.Context(), tenantScope, chi.URLParam(r, "id"), opportunity.Evidence{Kind: input.Kind, Summary: input.Summary, Confidence: input.Confidence}, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) scoreOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Score int `json:"score"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.ScoreOpportunity(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Score, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) transitionOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.TransitionOpportunity(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListAudit(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) reviewOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Decision  string `json:"decision"`
		Rationale string `json:"rationale"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.ReviewOpportunity(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Decision, input.Rationale, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createIncubation(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateIncubation(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Name, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listIncubations(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListIncubations(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) transitionIncubation(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.TransitionIncubation(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createBlueprint(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name                   string           `json:"name"`
		Description            string           `json:"description"`
		ValueProposition       string           `json:"value_proposition"`
		RequiredCapabilities   []string         `json:"required_capabilities"`
		ProductDefinitions     []map[string]any `json:"product_definitions"`
		WorkflowDefinitions    []map[string]any `json:"workflow_definitions"`
		MeteringDefinitions    []map[string]any `json:"metering_definitions"`
		PricingDefinitions     []map[string]any `json:"pricing_definitions"`
		ComplianceProfile      map[string]any   `json:"compliance_profile"`
		BusinessModel          map[string]any   `json:"business_model"`
		TargetMarketDefinition map[string]any   `json:"target_market_definition"`
		RevenueModel           map[string]any   `json:"revenue_model"`
		DeliveryModel          map[string]any   `json:"delivery_model"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateBlueprint(r.Context(), tenantScope, chi.URLParam(r, "id"), application.BlueprintInput{
		Name: input.Name, Description: input.Description, ValueProposition: input.ValueProposition,
		RequiredCapabilities: input.RequiredCapabilities, ProductDefinitions: input.ProductDefinitions,
		WorkflowDefinitions: input.WorkflowDefinitions, MeteringDefinitions: input.MeteringDefinitions,
		PricingDefinitions: input.PricingDefinitions, ComplianceProfile: input.ComplianceProfile,
		BusinessModel: input.BusinessModel, TargetMarketDefinition: input.TargetMarketDefinition,
		RevenueModel: input.RevenueModel, DeliveryModel: input.DeliveryModel,
	}, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listBlueprints(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListBlueprints(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) transitionBlueprint(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.TransitionBlueprint(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createCapability(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Definition  map[string]any `json:"definition"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateCapability(r.Context(), tenantScope, input.Name, input.Description, input.Definition, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listCapabilities(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListCapabilities(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateProvider(r.Context(), tenantScope, input.Name, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListProviders(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createProviderEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		CapabilityID   string `json:"capability_id"`
		AdapterType    string `json:"adapter_type"`
		AdapterVersion string `json:"adapter_version"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateProviderEndpoint(r.Context(), tenantScope, chi.URLParam(r, "id"), input.CapabilityID, input.AdapterType, input.AdapterVersion, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateProduct(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Name, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	items, err := s.store.ListProducts(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	item, err := s.store.GetProduct(r.Context(), tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, platform.Invalid("not_found", "product not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createProductVersion(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input application.ProductVersionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateProductVersion(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createSKU(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateSKU(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Code, input.Name, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createSKUVersion(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input application.SKUVersionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.CreateSKUVersion(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) publishProduct(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		ProductVersionID string `json:"product_version_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.store.PublishProduct(r.Context(), tenantScope, chi.URLParam(r, "id"), input.ProductVersionID, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transactionStore() (application.TransactionStore, error) {
	store, ok := s.store.(application.TransactionStore)
	if !ok {
		return nil, platform.Invalid("feature_unavailable", "transaction persistence is not configured")
	}
	return store, nil
}

func (s *Server) createQuote(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input application.QuoteInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateQuote(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listQuotes(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	items, err := store.ListQuotes(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getQuote(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.GetQuote(r.Context(), tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) transitionQuote(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionQuote(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		QuoteVersionID string `json:"quote_version_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateOrder(r.Context(), tenantScope, input.QuoteVersionID, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	items, err := store.ListOrders(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.GetOrder(r.Context(), tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) transitionOrder(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionOrder(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) transitionExecution(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input application.ExecutionTransitionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionExecution(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) transitionDelivery(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionDelivery(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) recordUsage(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		Quantity   int64     `json:"quantity"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RecordUsage(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Quantity, input.OccurredAt, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) recordProviderCost(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	var input struct {
		ProviderEndpointID string `json:"provider_endpoint_id"`
		Currency           string `json:"currency"`
		AmountMinor        int64  `json:"amount_minor"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RecordProviderCost(r.Context(), tenantScope, chi.URLParam(r, "id"), input.ProviderEndpointID, input.Currency, input.AmountMinor, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createCustomerCharge(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	store, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.CreateCustomerCharge(r.Context(), tenantScope, chi.URLParam(r, "id"), requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) financeStore() (application.FinanceStore, error) {
	store, ok := s.store.(application.FinanceStore)
	if !ok {
		return nil, platform.Invalid("feature_unavailable", "finance persistence is not configured")
	}
	return store, nil
}

func (s *Server) listFinance(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.financeStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.ListFinance(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func financeCommandContext(s *Server, w http.ResponseWriter, r *http.Request) (tenancy.Scope, string, application.FinanceStore, bool) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return tenancy.Scope{}, "", nil, false
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return tenancy.Scope{}, "", nil, false
	}
	store, err := s.financeStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return tenancy.Scope{}, "", nil, false
	}
	return tenantScope, requestID, store, true
}

func (s *Server) createWallet(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.WalletInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateWallet(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) postWalletAdjustment(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.WalletAdjustmentInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.PostWalletAdjustment(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) placeOrderHold(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.HoldInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.PlaceOrderHold(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) releaseHold(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ReleaseInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReleaseHold(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) postCustomerCharge(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ChargePostingInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.PostCustomerCharge(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) refundCustomerCharge(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.RefundInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RefundCustomerCharge(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createCommission(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CommissionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateCommission(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createProviderPayable(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	item, err := store.CreateProviderPayable(r.Context(), tenantScope, chi.URLParam(r, "id"), requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createSettlement(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SettlementInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSettlement(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) runReconciliation(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := financeCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ReconciliationInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RunReconciliation(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, err error) {
	code := "internal_error"
	message := "request failed"
	var domainErr *platform.Error
	if errors.As(err, &domainErr) {
		code = domainErr.Code
		message = domainErr.Message
	} else if status < 500 {
		message = err.Error()
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message, "request_id": r.Header.Get("X-Request-ID")}})
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]*rateBucket
}
type rateBucket struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, clients: map[string]*rateBucket{}}
}
func (l *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if index := strings.LastIndex(host, ":"); index > 0 {
			host = host[:index]
		}
		now := time.Now()
		l.mu.Lock()
		bucket := l.clients[host]
		if bucket == nil || now.After(bucket.reset) {
			bucket = &rateBucket{reset: now.Add(l.window)}
			l.clients[host] = bucket
		}
		bucket.count++
		remaining := l.limit - bucket.count
		allowed := bucket.count <= l.limit
		reset := bucket.reset
		l.mu.Unlock()
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		if !allowed {
			writeError(w, r, http.StatusTooManyRequests, platform.Invalid("rate_limited", "request rate limit exceeded"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
