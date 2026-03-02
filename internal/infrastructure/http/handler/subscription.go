package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/dto"
	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"go.uber.org/zap"
)

// SubscriptionHandler handles all subscription-related endpoints.
type SubscriptionHandler struct {
	getSubscription *usecase.GetSubscription
	acknowledge     *usecase.Acknowledge
	cancel          *usecase.Cancel
	refund          *usecase.Refund
	revoke          *usecase.Revoke
	deferSub        *usecase.DeferSubscription
	logger          *zap.Logger
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(
	getSubscription *usecase.GetSubscription,
	acknowledge *usecase.Acknowledge,
	cancel *usecase.Cancel,
	refund *usecase.Refund,
	revoke *usecase.Revoke,
	deferSub *usecase.DeferSubscription,
	logger *zap.Logger,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		getSubscription: getSubscription,
		acknowledge:     acknowledge,
		cancel:          cancel,
		refund:          refund,
		revoke:          revoke,
		deferSub:        deferSub,
		logger:          logger,
	}
}

// Get handles GET .../subscriptions/{subscriptionId}/tokens/{token}
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	resp, err := h.getSubscription.Execute(r.Context(), pkg, subID, token)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Acknowledge handles POST .../tokens/{token}:acknowledge
// Real Google Play API returns 204 No Content.
func (h *SubscriptionHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	if err := h.acknowledge.ExecuteSubscription(r.Context(), pkg, subID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Cancel handles POST .../tokens/{token}:cancel
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	if err := h.cancel.Execute(r.Context(), pkg, subID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Refund handles POST .../tokens/{token}:refund
func (h *SubscriptionHandler) Refund(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	if err := h.refund.Execute(r.Context(), pkg, subID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Revoke handles POST .../tokens/{token}:revoke
func (h *SubscriptionHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	if err := h.revoke.Execute(r.Context(), pkg, subID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Defer handles POST .../tokens/{token}:defer
// Real Google Play API returns {"newExpiryTimeMillis":"..."}.
func (h *SubscriptionHandler) Defer(w http.ResponseWriter, r *http.Request) {
	pkg, subID, token := pathParams(r)
	var body dto.DeferSubscriptionRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	resp, err := h.deferSub.Execute(r.Context(), pkg, subID, token, body)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func pathParams(r *http.Request) (pkg, subOrProductID, token string) {
	pkg = chi.URLParam(r, "packageName")
	subOrProductID = chi.URLParam(r, "subscriptionId")
	if subOrProductID == "" {
		subOrProductID = chi.URLParam(r, "productId")
	}
	token = chi.URLParam(r, "token")
	return
}
