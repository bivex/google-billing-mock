package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/dto"
	"go.uber.org/zap"
)

// OrderHandler handles the orders endpoints.
type OrderHandler struct {
	logger *zap.Logger
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(logger *zap.Logger) *OrderHandler {
	return &OrderHandler{logger: logger}
}

// Get handles GET .../orders/{orderId}
// Returns a synthetic order if orderId looks like a GPA token, 404 otherwise.
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")
	if !strings.HasPrefix(orderID, "GPA.") {
		writeError(w, http.StatusNotFound, "Order not found", "NOT_FOUND")
		return
	}
	writeJSON(w, http.StatusOK, syntheticOrder(orderID))
}

// BatchGet handles GET .../orders:batchGet
// Returns orders for all orderIds listed in the ?orderIds query param (comma-separated).
func (h *OrderHandler) BatchGet(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("orderIds")
	var orders []dto.OrderResponse
	if raw != "" {
		for _, oid := range strings.Split(raw, ",") {
			oid = strings.TrimSpace(oid)
			if strings.HasPrefix(oid, "GPA.") {
				orders = append(orders, syntheticOrder(oid))
			}
		}
	}
	if orders == nil {
		orders = []dto.OrderResponse{}
	}
	writeJSON(w, http.StatusOK, dto.BatchGetOrdersResponse{Orders: orders})
}

// Refund handles POST .../orders/{orderId}:refund
// Real Google Play API returns 204 No Content.
func (h *OrderHandler) Refund(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")
	if !strings.HasPrefix(orderID, "GPA.") {
		writeError(w, http.StatusNotFound, "Order not found", "NOT_FOUND")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// VoidedPurchasesList handles GET .../purchases/voidedpurchases
// Returns an empty list (mock has no voided purchases by default).
func (h *OrderHandler) VoidedPurchasesList(w http.ResponseWriter, r *http.Request) {
	resp := dto.VoidedPurchasesListResponse{
		PageInfo: dto.VoidedPageInfo{
			TotalResults:   0,
			StartIndex:     0,
			ResultsPerPage: 0,
		},
		TokenPagination: dto.VoidedTokenPagination{},
		VoidedPurchases: []dto.VoidedPurchase{},
	}
	writeJSON(w, http.StatusOK, resp)
}

func syntheticOrder(orderID string) dto.OrderResponse {
	return dto.OrderResponse{
		OrderId:       orderID,
		PurchaseToken: "",
		State:         "ORDER_STATE_CHARGE_ACCEPTED",
		CreateTime:    time.Now().UTC().Format(time.RFC3339),
	}
}
