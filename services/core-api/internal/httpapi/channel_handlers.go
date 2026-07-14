package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func (s *Server) channelStore() (application.ChannelStore, error) {
	store, ok := s.store.(application.ChannelStore)
	if !ok {
		return nil, platform.Invalid("feature_unavailable", "channel and marketplace persistence is not configured")
	}
	return store, nil
}

func channelCommandContext(s *Server, w http.ResponseWriter, r *http.Request) (tenancy.Scope, string, application.ChannelStore, bool) {
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
	store, err := s.channelStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return tenancy.Scope{}, "", nil, false
	}
	return tenantScope, requestID, store, true
}

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	store, err := s.channelStore()
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	item, err := store.ListChannels(r.Context(), tenantScope)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createResellerLevel(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ResellerLevelInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateResellerLevel(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createReseller(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ResellerInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateReseller(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createAttributionRule(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.AttributionRuleInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateAttributionRule(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) assignLeadOwnership(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.LeadOwnershipInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.AssignLeadOwnership(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createCustomerOwnership(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CustomerOwnershipInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateCustomerOwnership(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createTransferRequest(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.TransferRequestInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateTransferRequest(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) reviewTransfer(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ReviewInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewTransfer(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createCommissionRule(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CommissionRuleInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateCommissionRule(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) lockCommission(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.CommissionLockInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.LockCommission(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createSettlementCycle(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SettlementCycleInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSettlementCycle(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createSupplier(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SupplierInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSupplier(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) bindSupplierCapability(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SupplierCapabilityInput
	if !decode(w, r, &input) {
		return
	}
	input.SupplierID = chi.URLParam(r, "id")
	item, err := store.BindSupplierCapability(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) bindProviderSupplier(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ProviderSupplierInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.BindProviderSupplier(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createSupplierContract(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SupplierContractInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSupplierContract(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionSupplierContract(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionSupplierContract(r.Context(), scope, chi.URLParam(r, "id"), input.To, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) reviewSupplierContract(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ReviewInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewSupplierContract(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createSupplierRate(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SupplierRateInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateSupplierRate(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) recordSupplierQuality(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SupplierQualityInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RecordSupplierQuality(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createDeveloper(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.DeveloperInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateDeveloper(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createPublisher(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.PublisherInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreatePublisher(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createListing(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ListingInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateListing(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createListingVersion(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ListingVersionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateListingVersion(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) transitionListing(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := store.TransitionListing(r.Context(), scope, chi.URLParam(r, "id"), input.To, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) reviewListing(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ListingReviewInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewListing(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) runSandbox(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.SandboxRunInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RunSandbox(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) recordListingQuality(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ListingQualityInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RecordListingQuality(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) createMarketplaceDispute(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.MarketplaceDisputeInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.CreateMarketplaceDispute(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) resolveMarketplaceDispute(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.DisputeResolutionInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ResolveMarketplaceDispute(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) requestTakedown(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.TakedownInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.RequestTakedown(r.Context(), scope, input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) reviewTakedown(w http.ResponseWriter, r *http.Request) {
	scope, key, store, ok := channelCommandContext(s, w, r)
	if !ok {
		return
	}
	var input application.ReviewInput
	if !decode(w, r, &input) {
		return
	}
	item, err := store.ReviewTakedown(r.Context(), scope, chi.URLParam(r, "id"), input, key)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
