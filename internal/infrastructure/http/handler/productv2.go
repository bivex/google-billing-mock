package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"go.uber.org/zap"
)

// ProductV2Handler handles the productsv2 endpoint.
type ProductV2Handler struct {
	getV2  *usecase.GetProductV2
	logger *zap.Logger
}

// NewProductV2Handler creates a new ProductV2Handler.
func NewProductV2Handler(getV2 *usecase.GetProductV2, logger *zap.Logger) *ProductV2Handler {
	return &ProductV2Handler{getV2: getV2, logger: logger}
}

// Get handles GET .../productsv2/tokens/{token}
// Path has no productId — uses empty string for repo lookup (token-keyed).
func (h *ProductV2Handler) Get(w http.ResponseWriter, r *http.Request) {
	pkg := chi.URLParam(r, "packageName")
	token := chi.URLParam(r, "token")
	resp, err := h.getV2.Execute(r.Context(), pkg, "", token)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
