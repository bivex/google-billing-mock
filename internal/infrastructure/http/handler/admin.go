package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/bivex/google-billing-mock/internal/application/dto"
	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/domain/repository"
	"github.com/bivex/google-billing-mock/internal/infrastructure/mock"
	"go.uber.org/zap"
)

// AdminHandler manages /admin/* endpoints for scenario and purchase management.
type AdminHandler struct {
	repo        repository.PurchaseRepository
	scenarioMgr *mock.ScenarioManager
	scenariosPath string
	logger      *zap.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(repo repository.PurchaseRepository, scenarioMgr *mock.ScenarioManager, scenariosPath string, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{repo: repo, scenarioMgr: scenarioMgr, scenariosPath: scenariosPath, logger: logger}
}

// ListScenarios handles GET /admin/scenarios
func (h *AdminHandler) ListScenarios(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.scenarioMgr.ListScenarios())
}

// AddScenario handles POST /admin/scenarios
func (h *AdminHandler) AddScenario(w http.ResponseWriter, r *http.Request) {
	var s mock.ScenarioConfig
	if !decodeJSON(w, r, &s) {
		return
	}
	h.scenarioMgr.AddScenario(s)
	writeJSON(w, http.StatusCreated, s)
}

// DeleteScenario handles DELETE /admin/scenarios/{name}
func (h *AdminHandler) DeleteScenario(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !h.scenarioMgr.DeleteScenario(name) {
		writeError(w, http.StatusNotFound, "scenario not found", "NOT_FOUND")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReloadScenarios handles POST /admin/scenarios/reload — reloads from file.
func (h *AdminHandler) ReloadScenarios(w http.ResponseWriter, _ *http.Request) {
	if err := h.scenarioMgr.LoadFromFile(h.scenariosPath); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "INTERNAL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

// SeedSubscription handles POST /admin/purchases/subscriptions
func (h *AdminHandler) SeedSubscription(w http.ResponseWriter, r *http.Request) {
	var req dto.SeedSubscriptionRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	ps := entity.PaymentStateReceived
	if req.PaymentState != nil {
		ps = entity.PaymentState(*req.PaymentState)
	}
	purchase := &entity.SubscriptionPurchase{
		Token:                entity.PurchaseToken(req.Token),
		SubscriptionID:       entity.SubscriptionID(req.SubscriptionID),
		PackageName:          entity.PackageName(req.PackageName),
		Kind:                 "androidpublisher#subscriptionPurchase",
		PurchaseState:        entity.PurchaseState(req.PurchaseState),
		PaymentState:         &ps,
		AcknowledgementState: entity.AcknowledgementState(req.AcknowledgementState),
		AutoRenewing:         req.AutoRenewing,
		ExpiryTimeMillis:     req.ExpiryTimeMillis,
		PurchaseTimeMillis:   time.Now().UnixMilli(),
		RegionCode:           "US",
		OrderID:              "GPA.SEEDED-0000-0000-0000",
	}
	if req.CancelReason != nil {
		cr := entity.CancelReason(*req.CancelReason)
		purchase.CancelReason = &cr
	}
	_ = h.repo.SeedSubscription(entity.PurchaseToken(req.Token), purchase)
	writeJSON(w, http.StatusCreated, purchase)
}

// SeedProduct handles POST /admin/purchases/products
func (h *AdminHandler) SeedProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.SeedProductRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ps := entity.PaymentStateReceived
	purchase := &entity.ProductPurchase{
		Token:                entity.PurchaseToken(req.Token),
		ProductID:            entity.ProductID(req.ProductID),
		PackageName:          entity.PackageName(req.PackageName),
		Kind:                 "androidpublisher#productPurchase",
		PurchaseState:        entity.PurchaseState(req.PurchaseState),
		PaymentState:         &ps,
		AcknowledgementState: entity.AcknowledgementState(req.AcknowledgementState),
		PurchaseTimeMillis:   time.Now().UnixMilli(),
		RegionCode:           "US",
		OrderID:              "GPA.SEEDED-0000-0000-0001",
		Quantity:             1,
	}
	_ = h.repo.SeedProduct(entity.PurchaseToken(req.Token), purchase)
	writeJSON(w, http.StatusCreated, purchase)
}

// ListSubscriptions handles GET /admin/purchases/subscriptions
func (h *AdminHandler) ListSubscriptions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.ListSubscriptions())
}

// ListProducts handles GET /admin/purchases/products
func (h *AdminHandler) ListProducts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.ListProducts())
}
