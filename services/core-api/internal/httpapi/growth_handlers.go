package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func (s *Server) growthStore() (application.GrowthStore, error) {
	store, ok := s.store.(application.GrowthStore)
	if !ok {
		return nil, platform.Invalid("feature_unavailable", "growth persistence is not configured")
	}
	return store, nil
}

func growthCommandContext(s *Server, w http.ResponseWriter, r *http.Request) (tenancy.Scope, string, application.GrowthStore, bool) {
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
	store, err := s.growthStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return tenancy.Scope{}, "", nil, false
	}
	return tenantScope, requestID, store, true
}

func (s *Server) listGrowth(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.growthStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.ListGrowth(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createMarketSegment(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.MarketSegmentInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateMarketSegment(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createICPDefinition(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ICPDefinitionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateICPDefinition(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createLead(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.LeadInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateLead(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) addLeadEvidence(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.LeadEvidenceInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.AddLeadEvidence(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionLead(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionLead(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createContact(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ContactInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateContact(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createProofTemplate(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ProofTemplateInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateProofTemplate(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createProofRequest(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ProofRequestInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateProofRequest(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) generateProof(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ProofGenerationInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.GenerateProof(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) reviewProof(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ProofReviewInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewProof(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createCampaign(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CampaignInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateCampaign(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) addCampaignStep(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CampaignStepInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.AddCampaignStep(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionCampaign(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionCampaign(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) reviewCampaign(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CampaignApprovalInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewCampaign(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createSuppression(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SuppressionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSuppression(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) planOutreach(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.OutreachPlanInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.PlanOutreach(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionOutreach(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.OutreachTransitionInput
	if !decode(w, r, &input) {
		return
	}
	if input.To != "cancelled" {
		writeError(w, r, http.StatusLocked, platform.Invalid("outbound_delivery_disabled", "outbound delivery transitions require a disabled feature and a trusted adapter"))
		return
	}
	item, err := store.TransitionOutreach(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ConversationInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateConversation(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) addConversationMessage(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ConversationMessageInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.AddConversationMessage(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createDeal(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.DealInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateDeal(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) getDeal(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.growthStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.GetDeal(r.Context(), tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, platform.Invalid("not_found", "deal not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) transitionDeal(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionDeal(r.Context(), tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createDealQuote(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, growthStore, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	transactionStore, err := s.transactionStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	dealID := chi.URLParam(r, "id")
	deal, err := growthStore.GetDeal(r.Context(), tenantScope, dealID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, platform.Invalid("not_found", "deal not found"))
		return
	}
	var input application.QuoteInput
	if !decode(w, r, &input) {
		return
	}
	input.DealID = deal.ID
	input.CustomerID = deal.CustomerID
	if input.Currency == "" {
		input.Currency = deal.Currency
	}
	item, err := transactionStore.CreateQuote(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createExperiment(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ExperimentInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateExperiment(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionExperiment(w http.ResponseWriter, r *http.Request) {
	tenantScope, requestID, store, ok := growthCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ExperimentTransitionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionExperiment(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
