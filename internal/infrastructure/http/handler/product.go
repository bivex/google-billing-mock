package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"go.uber.org/zap"
)

// ProductHandler handles one-time product purchase endpoints.
type ProductHandler struct {
	getProduct  *usecase.GetProduct
	acknowledge *usecase.Acknowledge
	consume     *usecase.ConsumeProduct
	logger      *zap.Logger
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(getProduct *usecase.GetProduct, acknowledge *usecase.Acknowledge, consume *usecase.ConsumeProduct, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{getProduct: getProduct, acknowledge: acknowledge, consume: consume, logger: logger}
}

// Get handles GET .../products/{productId}/tokens/{token}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	productID := chi.URLParam(r, "productId")
	token := chi.URLParam(r, "token")

	resp, err := h.getProduct.Execute(r.Context(), pkg, productID, token)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Acknowledge handles POST .../products/{productId}/tokens/{token}:acknowledge
// Real Google Play API returns 204 No Content.
func (h *ProductHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	productID := chi.URLParam(r, "productId")
	token := chi.URLParam(r, "token")

	if err := h.acknowledge.ExecuteProduct(r.Context(), pkg, productID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Consume handles POST .../products/{productId}/tokens/{token}:consume
// Real Google Play API returns 204 No Content.
func (h *ProductHandler) Consume(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	productID := chi.URLParam(r, "productId")
	token := chi.URLParam(r, "token")

	if err := h.consume.Execute(r.Context(), pkg, productID, token); err != nil {
		mapError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
