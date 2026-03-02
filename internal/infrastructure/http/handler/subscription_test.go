package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bivex/google-billing-mock/internal/application/usecase"
	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/infrastructure/http/handler"
	"github.com/bivex/google-billing-mock/internal/infrastructure/mock"
	"go.uber.org/zap"
)

func setupSubscriptionHandler(t *testing.T) (*handler.SubscriptionHandler, *mock.InMemoryRepository) {
	t.Helper()
	sm := mock.NewScenarioManager()
	paymentState := 1
	sm.AddScenario(mock.ScenarioConfig{
		Name:                 "active",
		TokenPrefix:          "active_",
		Type:                 "subscription",
		PurchaseState:        0,
		PaymentState:         &paymentState,
		AcknowledgementState: 0,
		AutoRenewing:         true,
		ExpiryOffsetSeconds:  2592000,
	})
	errCode := 404
	sm.AddScenario(mock.ScenarioConfig{
		Name:         "invalid",
		TokenPrefix:  "invalid_",
		Type:         "subscription",
		ErrorCode:    &errCode,
		ErrorMessage: "Purchase token not found",
	})
	repo := mock.NewInMemoryRepository(sm)
	log := zap.NewNop()

	getSub := usecase.NewGetSubscription(repo, log)
	ack := usecase.NewAcknowledge(repo, log)
	cancel := usecase.NewCancel(repo, log)
	refund := usecase.NewRefund(repo, log)
	revoke := usecase.NewRevoke(repo, log)
	deferSub := usecase.NewDeferSubscription(repo, log)

	h := handler.NewSubscriptionHandler(getSub, ack, cancel, refund, revoke, deferSub, log)
	return h, repo
}

func chiRequest(method, path string, body string, params map[string]string) *http.Request {
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	r := httptest.NewRequest(method, path, reqBody)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSubscriptionHandler_Get_Success(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	r := chiRequest("GET", "/", "", map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "active_token",
	})
	w := httptest.NewRecorder()
	h.Get(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "androidpublisher#subscriptionPurchase", resp["kind"])
}

func TestSubscriptionHandler_Get_InvalidToken(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	r := chiRequest("GET", "/", "", map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "invalid_abc",
	})
	w := httptest.NewRecorder()
	h.Get(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubscriptionHandler_Get_UnknownToken(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	r := chiRequest("GET", "/", "", map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "unknown_xyz",
	})
	w := httptest.NewRecorder()
	h.Get(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubscriptionHandler_Acknowledge_Success(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	params := map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "active_ack",
	}
	r := chiRequest("POST", "/", "", params)
	w := httptest.NewRecorder()
	h.Acknowledge(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSubscriptionHandler_Acknowledge_AlreadyAcked(t *testing.T) {
	h, repo := setupSubscriptionHandler(t)
	// Pre-seed an already-acknowledged subscription
	ps := entity.PaymentStateReceived
	purchase := &entity.SubscriptionPurchase{
		Token:                "acked_tok",
		SubscriptionID:       "sub1",
		PackageName:          "com.test",
		Kind:                 "androidpublisher#subscriptionPurchase",
		PurchaseState:        entity.PurchaseStatePurchased,
		PaymentState:         &ps,
		AcknowledgementState: entity.AcknowledgementStateAcknowledged,
		ExpiryTimeMillis:     time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
		PurchaseTimeMillis:   time.Now().UnixMilli(),
	}
	_ = repo.SeedSubscription("acked_tok", purchase)

	r := chiRequest("POST", "/", "", map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "acked_tok",
	})
	w := httptest.NewRecorder()
	h.Acknowledge(w, r)
	// Acknowledging an already-acked purchase is a client error (500 from domain error)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSubscriptionHandler_Cancel(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	params := map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "active_cancel",
	}
	r := chiRequest("POST", "/", "", params)
	w := httptest.NewRecorder()
	h.Cancel(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubscriptionHandler_Revoke(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	params := map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "active_revoke",
	}
	r := chiRequest("POST", "/", "", params)
	w := httptest.NewRecorder()
	h.Revoke(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSubscriptionHandler_Defer_Success(t *testing.T) {
	h, _ := setupSubscriptionHandler(t)
	future := time.Now().Add(60 * 24 * time.Hour).UnixMilli()
	b, _ := json.Marshal(future)
	bodyStr := `{"deferralInfo":{"desiredExpiryTimeMillis":"` + strings.Trim(string(b), `"`) + `"}}`

	r := chiRequest("POST", "/", bodyStr, map[string]string{
		"packageName":    "com.test",
		"subscriptionId": "sub1",
		"token":          "active_defer",
	})
	w := httptest.NewRecorder()
	h.Defer(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
