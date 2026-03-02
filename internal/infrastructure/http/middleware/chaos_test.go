package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bivex/google-billing-mock/internal/infrastructure/http/middleware"
)

// okHandler is a simple 200 OK handler used as the "next" in the chain.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ── Latency ──────────────────────────────────────────────────────────────────

func TestChaos_NoLatency_NoDelay(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(w, r)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, elapsed, 50*time.Millisecond, "should be fast with zero latency")
}

func TestChaos_ConfigLatency_AddsDelay(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{DefaultLatencyMs: 100})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(w, r)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond, "should honour config latency")
}

func TestChaos_HeaderLatencyOverride(t *testing.T) {
	// Config = 0, header overrides to 150ms
	h := middleware.Chaos(middleware.ChaosConfig{})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Mock-Latency-Ms", "150")
	w := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(w, r)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond)
}

func TestChaos_InvalidLatencyHeader_Ignored(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{DefaultLatencyMs: 0})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Mock-Latency-Ms", "not-a-number")
	w := httptest.NewRecorder()

	start := time.Now()
	h.ServeHTTP(w, r)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, elapsed, 50*time.Millisecond, "invalid header should be ignored")
}

// ── Error injection ───────────────────────────────────────────────────────────

func TestChaos_ErrorRate1_AlwaysReturns500(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 1.0})(okHandler)

	for i := 0; i < 10; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "iteration %d", i)

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		errObj, ok := body["error"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(500), errObj["code"])
		assert.Equal(t, "INTERNAL", errObj["status"])
	}
}

func TestChaos_ErrorRate0_NeverReturns500(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 0.0})(okHandler)

	for i := 0; i < 20; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code, "iteration %d should be 200", i)
	}
}

func TestChaos_HeaderErrorRateOverride_AlwaysErrors(t *testing.T) {
	// Config = 0, header forces 100% error rate
	h := middleware.Chaos(middleware.ChaosConfig{})(okHandler)

	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Mock-Error-Rate", "1.0")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "iteration %d", i)
	}
}

func TestChaos_HeaderErrorRateZero_OverridesConfigError(t *testing.T) {
	// Config = 1.0 (always error), header overrides to 0 (never error)
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 1.0})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Mock-Error-Rate", "0.0")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChaos_InvalidErrorRateHeader_Ignored(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 0.0})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Mock-Error-Rate", "bad-value")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Error response body ───────────────────────────────────────────────────────

func TestChaos_ErrorResponse_IsJSON(t *testing.T) {
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 1.0})(okHandler)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	_, hasError := body["error"]
	assert.True(t, hasError)
}

// ── Next handler NOT called on injected error ─────────────────────────────────

func TestChaos_NextNotCalledOnInjectedError(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := middleware.Chaos(middleware.ChaosConfig{ErrorRate: 1.0})(next)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.False(t, called, "next handler must not be called when error is injected")
}
