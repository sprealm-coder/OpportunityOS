package httpapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

func (s *Server) integrationStore() (application.IntegrationStore, error) {
	store, ok := s.store.(application.IntegrationStore)
	if !ok {
		return nil, platform.Invalid("integration_unavailable", "phase H integration store is unavailable")
	}
	return store, nil
}

func (s *Server) listIntelligence(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	result, err := store.ListIntelligence(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createSource(w http.ResponseWriter, r *http.Request) {
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
	var input application.SourceInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.CreateSource(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) importSignal(w http.ResponseWriter, r *http.Request) {
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
	var input application.SignalInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.ImportSignal(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) promoteSignal(w http.ResponseWriter, r *http.Request) {
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
	var input application.SignalPromotionInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.PromoteSignal(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	result, err := store.ListAnalytics(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) recordOutcomeFeedback(w http.ResponseWriter, r *http.Request) {
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
	var input application.OutcomeFeedbackInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.RecordOutcomeFeedback(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) registerAdapterIdentity(w http.ResponseWriter, r *http.Request) {
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
	var input application.AdapterIdentityInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.RegisterAdapterIdentity(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) startWorkflowRun(w http.ResponseWriter, r *http.Request) {
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
	var input application.WorkflowRunInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.StartWorkflowRun(r.Context(), tenantScope, chi.URLParam(r, "id"), input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) leaseWorkflowStep(w http.ResponseWriter, r *http.Request) {
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
	var input application.WorkflowLeaseInput
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.LeaseWorkflowStep(r.Context(), tenantScope, input, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) ingestAdapterResult(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, platform.Invalid("invalid_adapter_result", err.Error()))
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.IngestAdapterResult(r.Context(), application.AdapterIngressRequest{
		KeyID: strings.TrimSpace(r.Header.Get("X-Adapter-Key")), Timestamp: strings.TrimSpace(r.Header.Get("X-Adapter-Timestamp")),
		Nonce: strings.TrimSpace(r.Header.Get("X-Adapter-Nonce")), Signature: strings.TrimSpace(r.Header.Get("X-Adapter-Signature")), Body: body,
	})
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, platform.Invalid("adapter_result_rejected", "adapter authentication or workflow lease validation failed"))
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listOperations(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	result, err := store.ListOperations(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) replayOutbox(w http.ResponseWriter, r *http.Request) {
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
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.ReplayOutbox(r.Context(), tenantScope, chi.URLParam(r, "id"), input.Reason, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) acknowledgeOperationalAlert(w http.ResponseWriter, r *http.Request) {
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
	store, err := s.integrationStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.AcknowledgeOperationalAlert(r.Context(), tenantScope, chi.URLParam(r, "id"), requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
