package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"go.uber.org/zap"
)

// SubscriptionV2Handler handles the subscriptionsv2 endpoints.
// Note: paths have no subscriptionId, only packageName + token.
type SubscriptionV2Handler struct {
	getV2    *usecase.GetSubscriptionV2
	cancel   *usecase.Cancel
	revoke   *usecase.Revoke
	deferV2  *usecase.DeferSubscriptionV2
	logger   *zap.Logger
}

// NewSubscriptionV2Handler creates a new SubscriptionV2Handler.
func NewSubscriptionV2Handler(
	getV2 *usecase.GetSubscriptionV2,
	cancel *usecase.Cancel,
	revoke *usecase.Revoke,
	deferV2 *usecase.DeferSubscriptionV2,
	logger *zap.Logger,
) *SubscriptionV2Handler {
	return &SubscriptionV2Handler{
		getV2:   getV2,
		cancel:  cancel,
		revoke:  revoke,
		deferV2: deferV2,
		logger:  logger,
	}
}

// Get handles GET .../subscriptionsv2/tokens/{token}
func (h *SubscriptionV2Handler) Get(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	token := chi.URLParam(r, "token")
	resp, err := h.getV2.Execute(r.Context(), pkg, "", token)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Cancel handles POST .../subscriptionsv2/tokens/{token}:cancel
// Returns 200 with empty body (CancelSubscriptionPurchaseResponse is empty).
func (h *SubscriptionV2Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	token := chi.URLParam(r, "token")
	if err := h.cancel.Execute(r.Context(), pkg, "", token); err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// Revoke handles POST .../subscriptionsv2/tokens/{token}:revoke
// Returns 200 with empty body (RevokeSubscriptionPurchaseResponse is empty).
func (h *SubscriptionV2Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	token := chi.URLParam(r, "token")
	if err := h.revoke.Execute(r.Context(), pkg, "", token); err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// Defer handles POST .../subscriptionsv2/tokens/{token}:defer
// Extends expiry by 30 days and returns DeferSubscriptionPurchaseResponse.
func (h *SubscriptionV2Handler) Defer(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	token := chi.URLParam(r, "token")
	resp, err := h.deferV2.Execute(r.Context(), pkg, token)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
